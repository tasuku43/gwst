package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gws/internal/core/gitcmd"
	"github.com/tasuku43/gws/internal/core/output"
	"github.com/tasuku43/gws/internal/core/paths"
	"github.com/tasuku43/gws/internal/domain/repo"
	"github.com/tasuku43/gws/internal/domain/repospec"
	"github.com/tasuku43/gws/internal/domain/template"
	"github.com/tasuku43/gws/internal/domain/workspace"
	"github.com/tasuku43/gws/internal/ops/doctor"
	"github.com/tasuku43/gws/internal/ops/initcmd"
	"github.com/tasuku43/gws/internal/ui"
)

const defaultRepoProtocol = "ssh"

// Run is a placeholder for the CLI entrypoint.
func Run() error {
	fs := flag.NewFlagSet("gws", flag.ContinueOnError)
	var rootFlag string
	var noPrompt bool
	verboseFlag := envBool("GWS_VERBOSE")
	var helpFlag bool
	fs.StringVar(&rootFlag, "root", "", "override gws root")
	fs.BoolVar(&noPrompt, "no-prompt", false, "disable interactive prompt")
	fs.BoolVar(&verboseFlag, "verbose", verboseFlag, "show detailed logs")
	fs.BoolVar(&verboseFlag, "v", verboseFlag, "show detailed logs")
	fs.BoolVar(&helpFlag, "help", false, "show help")
	fs.BoolVar(&helpFlag, "h", false, "show help")
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		printGlobalHelp(os.Stdout)
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	gitcmd.SetVerbose(verboseFlag)

	args := fs.Args()
	if helpFlag {
		if len(args) > 0 && printCommandHelp(args[0], os.Stdout) {
			return nil
		}
		printGlobalHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		printGlobalHelp(os.Stdout)
		return nil
	}
	if args[0] == "help" {
		if len(args) > 1 && printCommandHelp(args[1], os.Stdout) {
			return nil
		}
		printGlobalHelp(os.Stdout)
		return nil
	}

	rootDir, err := paths.ResolveRoot(rootFlag)
	if err != nil {
		return err
	}

	ctx := context.Background()
	switch args[0] {
	case "init":
		return runInit(rootDir, args[1:])
	case "doctor":
		return runDoctor(ctx, rootDir, args[1:])
	case "repo":
		return runRepo(ctx, rootDir, args[1:])
	case "template":
		return runTemplate(ctx, rootDir, args[1:], noPrompt)
	case "new":
		return runWorkspaceNew(ctx, rootDir, args[1:], noPrompt)
	case "review":
		return runReview(ctx, rootDir, args[1:], noPrompt)
	case "issue":
		return runIssue(ctx, rootDir, args[1:], noPrompt)
	case "add":
		return runWorkspaceAdd(ctx, rootDir, args[1:])
	case "ls":
		return runWorkspaceList(ctx, rootDir, args[1:])
	case "status":
		return runWorkspaceStatus(ctx, rootDir, args[1:])
	case "rm":
		return runWorkspaceRemove(ctx, rootDir, args[1:])
	case "open":
		return runWorkspaceOpen(ctx, rootDir, args[1:], noPrompt)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runInit(rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printInitHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gws init")
	}
	result, err := initcmd.Run(rootDir)
	if err != nil {
		return err
	}
	writeInitText(result)
	return nil
}

func envBool(key string) bool {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return false
	}
	switch strings.ToLower(val) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func runTemplate(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printTemplateHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "ls":
		return runTemplateList(ctx, rootDir, args[1:])
	case "new":
		return runTemplateNew(ctx, rootDir, args[1:], noPrompt)
	default:
		return fmt.Errorf("unknown template subcommand: %s", args[0])
	}
}

func runTemplateList(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printTemplateLsHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gws template ls")
	}
	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}
	names := template.Names(file)
	writeTemplateListText(file, names)
	return nil
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func runTemplateNew(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	newFlags := flag.NewFlagSet("template new", flag.ContinueOnError)
	var helpFlag bool
	var repos stringSliceFlag
	newFlags.Var(&repos, "repo", "repo spec (repeatable)")
	newFlags.BoolVar(&helpFlag, "help", false, "show help")
	newFlags.BoolVar(&helpFlag, "h", false, "show help")
	newFlags.SetOutput(os.Stdout)
	newFlags.Usage = func() {
		printTemplateNewHelp(os.Stdout)
	}
	if err := newFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printTemplateNewHelp(os.Stdout)
		return nil
	}
	if newFlags.NArg() > 1 {
		return fmt.Errorf("usage: gws template new [<name>] [--repo <repo> ...]")
	}

	name := ""
	if newFlags.NArg() == 1 {
		name = newFlags.Arg(0)
	}

	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}

	repoSpecs := template.NormalizeRepos(repos)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	if strings.TrimSpace(name) == "" && len(repoSpecs) == 0 {
		if noPrompt {
			return fmt.Errorf("template name and repos are required with --no-prompt")
		}
		choices, err := buildTemplateRepoChoices(rootDir)
		if err != nil {
			return err
		}
		if len(choices) == 0 {
			return fmt.Errorf("no repos found; run gws repo get first")
		}
		name, repoSpecs, err = ui.PromptTemplateRepos("gws template new", name, choices, theme, useColor)
		if err != nil {
			return err
		}
		repoSpecs = template.NormalizeRepos(repoSpecs)
	} else {
		if strings.TrimSpace(name) == "" {
			if noPrompt {
				return fmt.Errorf("template name is required with --no-prompt")
			}
			name, err = ui.PromptTemplateName("gws template new", "", theme, useColor)
			if err != nil {
				return err
			}
		}
		if len(repoSpecs) == 0 {
			if noPrompt {
				return fmt.Errorf("repos are required with --no-prompt")
			}
			choices, err := buildTemplateRepoChoices(rootDir)
			if err != nil {
				return err
			}
			if len(choices) == 0 {
				return fmt.Errorf("no repos found; run gws repo get first")
			}
			var selected []string
			name, selected, err = ui.PromptTemplateRepos("gws template new", name, choices, theme, useColor)
			if err != nil {
				return err
			}
			repoSpecs = template.NormalizeRepos(selected)
		}
	}

	if err := template.ValidateName(name); err != nil {
		return err
	}
	if _, exists := file.Templates[name]; exists {
		return fmt.Errorf("template already exists: %s", name)
	}

	if len(repoSpecs) == 0 {
		return fmt.Errorf("at least one repo is required")
	}

	for _, repoSpec := range repoSpecs {
		if _, err := repospec.Normalize(repoSpec); err != nil {
			return err
		}
		if _, exists, err := repo.Exists(rootDir, repoSpec); err != nil {
			return err
		} else if !exists {
			return fmt.Errorf("repo store not found, run: gws repo get %s", repoSpec)
		}
	}

	if file.Templates == nil {
		file.Templates = map[string]template.Template{}
	}
	file.Templates[name] = template.Template{Repos: repoSpecs}

	if err := template.Save(rootDir, file); err != nil {
		return err
	}

	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	renderer.Section("Result")
	renderer.Bullet(name)
	var reposDisplay []string
	for _, repoSpec := range repoSpecs {
		reposDisplay = append(reposDisplay, displayTemplateRepo(repoSpec))
	}
	renderTreeLines(renderer, reposDisplay, treeLineNormal)
	return nil
}

func buildTemplateRepoChoices(rootDir string) ([]ui.PromptChoice, error) {
	repos, _, err := repo.List(rootDir)
	if err != nil {
		return nil, err
	}
	var choices []ui.PromptChoice
	for _, entry := range repos {
		label := displayRepoKey(entry.RepoKey)
		value := repoSpecFromKey(entry.RepoKey)
		choices = append(choices, ui.PromptChoice{Label: label, Value: value})
	}
	return choices, nil
}

func runDoctor(ctx context.Context, rootDir string, args []string) error {
	doctorFlags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	var fix bool
	var helpFlag bool
	doctorFlags.SetOutput(os.Stdout)
	doctorFlags.Usage = func() {
		printDoctorHelp(os.Stdout)
	}
	doctorFlags.BoolVar(&fix, "fix", false, "remove stale locks only")
	doctorFlags.BoolVar(&helpFlag, "help", false, "show help")
	doctorFlags.BoolVar(&helpFlag, "h", false, "show help")
	if err := doctorFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printDoctorHelp(os.Stdout)
		return nil
	}
	if doctorFlags.NArg() != 0 {
		return fmt.Errorf("usage: gws doctor [--fix]")
	}
	now := time.Now().UTC()
	if fix {
		result, err := doctor.Fix(ctx, rootDir, now)
		if err != nil {
			return err
		}
		writeDoctorText(result.Result, result.Fixed)
		return nil
	}

	result, err := doctor.Check(ctx, rootDir, now)
	if err != nil {
		return err
	}
	writeDoctorText(result, nil)
	return nil
}

func runRepo(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printRepoHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "get":
		return runRepoGet(ctx, rootDir, args[1:])
	case "ls":
		return runRepoList(ctx, rootDir, args[1:])
	default:
		return fmt.Errorf("unknown repo subcommand: %s", args[0])
	}
}

func runRepoGet(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 0 || (len(args) == 1 && isHelpArg(args[0])) {
		printRepoGetHelp(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gws repo get <repo>")
	}
	repoSpec := strings.TrimSpace(args[0])
	if repoSpec == "" {
		return fmt.Errorf("repo is required")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	startSteps(renderer)
	output.Step(formatStep("repo get", displayRepoSpec(repoSpec), repoDestForSpec(rootDir, repoSpec)))

	store, err := repo.Get(ctx, rootDir, repoSpec)
	if err != nil {
		return err
	}
	renderer.Blank()
	renderer.Section("Result")
	renderer.Result(fmt.Sprintf("%s\t%s", store.RepoKey, store.StorePath))
	renderSuggestion(renderer, useColor, repoSrcAbs(rootDir, repoSpec))
	return nil
}

func runRepoList(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printRepoLsHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gws repo ls")
	}
	entries, warnings, err := repo.List(rootDir)
	if err != nil {
		return err
	}
	writeRepoListText(entries, warnings)
	return nil
}

func runWorkspaceNew(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	newFlags := flag.NewFlagSet("new", flag.ContinueOnError)
	var templateName string
	var helpFlag bool
	newFlags.StringVar(&templateName, "template", "", "template name")
	newFlags.BoolVar(&helpFlag, "help", false, "show help")
	newFlags.BoolVar(&helpFlag, "h", false, "show help")
	newFlags.SetOutput(os.Stdout)
	newFlags.Usage = func() {
		printNewHelp(os.Stdout)
	}
	if err := newFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printNewHelp(os.Stdout)
		return nil
	}
	if newFlags.NArg() > 1 {
		return fmt.Errorf("usage: gws new [--template <name>] [<WORKSPACE_ID>]")
	}

	workspaceID := ""
	if newFlags.NArg() == 1 {
		workspaceID = newFlags.Arg(0)
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	if templateName == "" || workspaceID == "" {
		if noPrompt {
			return fmt.Errorf("template name and workspace id are required without prompt")
		}
		var err error
		templateName, workspaceID, err = promptTemplateAndID(rootDir, templateName, workspaceID, theme, useColor)
		if err != nil {
			return err
		}
	}

	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}
	tmpl, ok := file.Templates[templateName]
	if !ok {
		return fmt.Errorf("template not found: %s", templateName)
	}
	missing, err := preflightTemplateRepos(ctx, rootDir, tmpl)
	if err != nil {
		return err
	}
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	var branches []string
	if !noPrompt {
		branches, err = promptTemplateBranches(ctx, tmpl, workspaceID, theme, useColor)
		if err != nil {
			return err
		}
	}

	startSteps(renderer)
	if err := ensureRepoGet(ctx, rootDir, missing, noPrompt, theme, useColor); err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, filepath.Join(rootDir, "workspaces", workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}

	if err := applyTemplate(ctx, rootDir, workspaceID, tmpl, branches); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("apply template failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, repos)
	renderSuggestion(renderer, useColor, wsDir)
	return nil
}

func runWorkspaceAdd(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printAddHelp(os.Stdout)
		return nil
	}
	if len(args) > 2 {
		return fmt.Errorf("usage: gws add [<WORKSPACE_ID>] [<repo>]")
	}
	workspaceID := ""
	repoSpec := ""
	if len(args) >= 1 {
		workspaceID = args[0]
	}
	if len(args) == 2 {
		repoSpec = args[1]
	}
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	if workspaceID == "" || repoSpec == "" {
		workspaces, wsWarn, err := workspace.List(rootDir)
		if err != nil {
			return err
		}
		if len(wsWarn) > 0 {
			// ignore warnings for selection
		}
		workspaceChoices := buildWorkspaceChoices(ctx, workspaces)
		if len(workspaceChoices) == 0 {
			return fmt.Errorf("no workspaces found")
		}

		repos, _, err := repo.List(rootDir)
		if err != nil {
			return err
		}
		var repoChoices []ui.PromptChoice
		for _, entry := range repos {
			label := displayRepoKey(entry.RepoKey)
			value := repoSpecFromKey(entry.RepoKey)
			repoChoices = append(repoChoices, ui.PromptChoice{Label: label, Value: value})
		}
		if len(repoChoices) == 0 {
			return fmt.Errorf("no repos found")
		}

		if workspaceID == "" || repoSpec == "" {
			workspaceID, repoSpec, err = ui.PromptWorkspaceAndRepo("gws add", workspaceChoices, repoChoices, workspaceID, repoSpec, theme, useColor)
			if err != nil {
				return err
			}
		}
	}

	startSteps(renderer)
	output.Step(formatStep("worktree add", displayRepoName(repoSpec), worktreeDest(rootDir, workspaceID, repoSpec)))

	if _, err := workspace.Add(ctx, rootDir, workspaceID, repoSpec, "", false); err != nil {
		return err
	}
	wsDir := filepath.Join(rootDir, "workspaces", workspaceID)
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderer.Blank()
	renderer.Section("Result")
	renderWorkspaceBlock(renderer, workspaceID, repos)
	renderSuggestion(renderer, useColor, filepath.Join(rootDir, "workspaces", workspaceID))
	return nil
}

func runIssue(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	issueFlags := flag.NewFlagSet("issue", flag.ContinueOnError)
	var workspaceID string
	var branch string
	var baseRef string
	var helpFlag bool
	issueFlags.StringVar(&workspaceID, "workspace-id", "", "workspace id")
	issueFlags.StringVar(&branch, "branch", "", "branch name")
	issueFlags.StringVar(&baseRef, "base", "", "base ref")
	issueFlags.BoolVar(&helpFlag, "help", false, "show help")
	issueFlags.BoolVar(&helpFlag, "h", false, "show help")
	issueFlags.SetOutput(os.Stdout)
	issueFlags.Usage = func() {
		printIssueHelp(os.Stdout)
	}
	if err := issueFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printIssueHelp(os.Stdout)
		return nil
	}

	workspaceID = strings.TrimSpace(workspaceID)
	branch = strings.TrimSpace(branch)
	baseRef = strings.TrimSpace(baseRef)

	if issueFlags.NArg() == 0 {
		if noPrompt {
			return fmt.Errorf("issue URL is required when --no-prompt is set")
		}
		if workspaceID != "" || branch != "" || baseRef != "" {
			return fmt.Errorf("--workspace-id, --branch, and --base are only valid when an issue URL is provided")
		}
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("interactive issue picker requires a TTY")
		}
		return runIssuePicker(ctx, rootDir, noPrompt)
	}

	if issueFlags.NArg() != 1 {
		return fmt.Errorf("usage: gws issue [<ISSUE_URL>] [--workspace-id <id>] [--branch <name>] [--base <ref>]")
	}

	raw := strings.TrimSpace(issueFlags.Arg(0))
	if raw == "" {
		return fmt.Errorf("issue URL is required")
	}

	req, err := parseIssueURL(raw)
	if err != nil {
		return err
	}
	repoURL := buildRepoURLFromParts(req.Host, req.Owner, req.Repo)

	branchProvided := branch != ""
	if workspaceID == "" {
		workspaceID = fmt.Sprintf("ISSUE-%d", req.Number)
	}
	if branch == "" {
		branch = fmt.Sprintf("issue/%d", req.Number)
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	if !noPrompt && !branchProvided {
		value, err := ui.PromptInputInline("branch", branch, nil, theme, useColor)
		if err != nil {
			return err
		}
		branch = strings.TrimSpace(value)
		if branch == "" {
			branch = fmt.Sprintf("issue/%d", req.Number)
		}
	}

	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Inputs")
	renderer.Bullet(fmt.Sprintf("provider: %s (%s)", strings.ToLower(req.Provider), req.Host))
	renderer.Bullet(fmt.Sprintf("repo: %s/%s", req.Owner, req.Repo))
	renderer.Bullet(fmt.Sprintf("issue: #%d", req.Number))
	renderer.Bullet(fmt.Sprintf("branch: %s", branch))
	if baseRef != "" {
		renderer.Bullet(fmt.Sprintf("base: %s", baseRef))
	}
	renderer.Blank()
	renderer.Section("Steps")

	_, exists, err := repo.Exists(rootDir, repoURL)
	if err != nil {
		return err
	}
	if !exists {
		if err := ensureRepoGet(ctx, rootDir, []string{repoURL}, noPrompt, theme, useColor); err != nil {
			return err
		}
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, filepath.Join(rootDir, "workspaces", workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}

	output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
	if _, err := workspace.AddWithBranch(ctx, rootDir, workspaceID, repoURL, "", branch, baseRef, true); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("issue setup failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, repos)
	renderSuggestion(renderer, useColor, wsDir)
	return nil
}

type issueRepoChoice struct {
	Label    string
	Value    string
	Provider string
	Host     string
	Owner    string
	Repo     string
}

type issueSummary struct {
	Number int
	Title  string
}

func runIssuePicker(ctx context.Context, rootDir string, noPrompt bool) error {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	repoChoices, err := buildIssueRepoChoices(rootDir)
	if err != nil {
		return err
	}
	if len(repoChoices) == 0 {
		return fmt.Errorf("no repos with supported hosts found")
	}

	promptChoices := make([]ui.PromptChoice, 0, len(repoChoices))
	repoByValue := make(map[string]issueRepoChoice, len(repoChoices))
	for _, choice := range repoChoices {
		promptChoices = append(promptChoices, ui.PromptChoice{Label: choice.Label, Value: choice.Value})
		repoByValue[choice.Value] = choice
	}

	repoSpec, err := ui.PromptChoiceSelect("gws issue", "repo", promptChoices, theme, useColor)
	if err != nil {
		return err
	}
	selectedRepo, ok := repoByValue[repoSpec]
	if !ok {
		return fmt.Errorf("selected repo not found")
	}
	if strings.ToLower(selectedRepo.Provider) != "github" {
		return fmt.Errorf("issue picker supports GitHub only for now: %s", selectedRepo.Host)
	}

	issues, err := fetchGitHubIssues(ctx, selectedRepo.Host, selectedRepo.Owner, selectedRepo.Repo)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		return fmt.Errorf("no issues found")
	}

	var issueChoices []ui.PromptChoice
	for _, issue := range issues {
		label := fmt.Sprintf("#%d", issue.Number)
		if strings.TrimSpace(issue.Title) != "" {
			label = fmt.Sprintf("#%d %s", issue.Number, strings.TrimSpace(issue.Title))
		}
		issueChoices = append(issueChoices, ui.PromptChoice{
			Label: label,
			Value: strconv.Itoa(issue.Number),
		})
	}

	selectedIssues, err := ui.PromptMultiSelect("gws issue", "issue", issueChoices, theme, useColor)
	if err != nil {
		return err
	}

	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Inputs")
	renderer.Bullet(fmt.Sprintf("provider: %s (%s)", strings.ToLower(selectedRepo.Provider), selectedRepo.Host))
	renderer.Bullet(fmt.Sprintf("repo: %s/%s", selectedRepo.Owner, selectedRepo.Repo))
	renderer.Bullet(fmt.Sprintf("issues: %s", formatIssueList(selectedIssues)))
	renderer.Blank()
	renderer.Section("Steps")

	_, exists, err := repo.Exists(rootDir, repoSpec)
	if err != nil {
		return err
	}
	if !exists {
		if err := ensureRepoGet(ctx, rootDir, []string{repoSpec}, noPrompt, theme, useColor); err != nil {
			return err
		}
	}

	repoURL := buildRepoURLFromParts(selectedRepo.Host, selectedRepo.Owner, selectedRepo.Repo)
	type issueWorkspaceResult struct {
		workspaceID string
		repos       []workspace.Repo
	}
	var results []issueWorkspaceResult
	var failure error
	var failureID string

	for _, value := range selectedIssues {
		num, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			failure = fmt.Errorf("invalid issue number: %s", value)
			failureID = value
			break
		}
		workspaceID := fmt.Sprintf("ISSUE-%d", num)
		branch := fmt.Sprintf("issue/%d", num)
		output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, filepath.Join(rootDir, "workspaces", workspaceID))))
		wsDir, err := workspace.New(ctx, rootDir, workspaceID)
		if err != nil {
			failure = err
			failureID = workspaceID
			break
		}

		output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
		if _, err := workspace.AddWithBranch(ctx, rootDir, workspaceID, repoURL, "", branch, "", true); err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = fmt.Errorf("issue setup failed: %w (rollback failed: %v)", err, rollbackErr)
			} else {
				failure = err
			}
			failureID = workspaceID
			break
		}

		repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
		results = append(results, issueWorkspaceResult{
			workspaceID: workspaceID,
			repos:       repos,
		})
	}

	if len(results) > 0 {
		renderer.Blank()
		renderer.Section("Result")
		for _, result := range results {
			renderWorkspaceBlock(renderer, result.workspaceID, result.repos)
		}
	}
	if failure != nil {
		return fmt.Errorf("%s: %w", failureID, failure)
	}
	return nil
}

func buildIssueRepoChoices(rootDir string) ([]issueRepoChoice, error) {
	repos, _, err := repo.List(rootDir)
	if err != nil {
		return nil, err
	}
	var choices []issueRepoChoice
	for _, entry := range repos {
		repoKey := displayRepoKey(entry.RepoKey)
		parts := strings.Split(repoKey, "/")
		if len(parts) < 3 {
			continue
		}
		host := parts[0]
		owner := parts[1]
		repoName := parts[2]
		provider := issueProviderForHost(host)
		label := fmt.Sprintf("%s (%s)", repoName, repoKey)
		value := repoSpecFromKey(entry.RepoKey)
		choices = append(choices, issueRepoChoice{
			Label:    label,
			Value:    value,
			Provider: provider,
			Host:     host,
			Owner:    owner,
			Repo:     repoName,
		})
	}
	return choices, nil
}

func issueProviderForHost(host string) string {
	lower := strings.ToLower(strings.TrimSpace(host))
	if strings.Contains(lower, "gitlab") {
		return "gitlab"
	}
	if strings.Contains(lower, "bitbucket") {
		return "bitbucket"
	}
	return "github"
}

type githubIssueItem struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	PullRequest json.RawMessage `json:"pull_request"`
}

func fetchGitHubIssues(ctx context.Context, host, owner, repoName string) ([]issueSummary, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" {
		return nil, fmt.Errorf("owner/repo is required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/issues", owner, repoName)
	args := []string{"api", "-X", "GET", endpoint, "-f", "state=open", "-f", "sort=updated", "-f", "direction=desc", "-f", "per_page=50"}
	if host != "" && !strings.EqualFold(host, "github.com") {
		args = append([]string{"api", "--hostname", host}, args[1:]...)
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("gh api failed: %s", msg)
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}
	return parseGitHubIssues(stdout.Bytes())
}

func parseGitHubIssues(data []byte) ([]issueSummary, error) {
	var raw []githubIssueItem
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gh api response: %w", err)
	}
	var issues []issueSummary
	for _, item := range raw {
		if item.Number == 0 {
			continue
		}
		if len(item.PullRequest) != 0 {
			continue
		}
		issues = append(issues, issueSummary{
			Number: item.Number,
			Title:  strings.TrimSpace(item.Title),
		})
	}
	return issues, nil
}

func formatIssueList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	var out []string
	for _, value := range values {
		val := strings.TrimSpace(value)
		if val == "" {
			continue
		}
		out = append(out, fmt.Sprintf("#%s", val))
	}
	return strings.Join(out, ", ")
}

func runReview(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || (len(args) == 1 && isHelpArg(args[0])) {
		printReviewHelp(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gws review <PR URL>")
	}
	raw := strings.TrimSpace(args[0])
	if raw == "" {
		return fmt.Errorf("PR URL is required")
	}

	req, err := parsePRURL(raw)
	if err != nil {
		return err
	}
	repoURL := buildRepoURL(req)

	_, exists, err := repo.Exists(rootDir, repoURL)
	if err != nil {
		return err
	}
	workspaceID := fmt.Sprintf("REVIEW-PR-%d", req.Number)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Info")
	renderer.Bullet(fmt.Sprintf("provider: %s (%s)", strings.ToLower(req.Provider), req.Host))
	renderer.Bullet("fork PRs supported (fetches PR ref directly)")
	renderer.Blank()
	renderer.Section("Steps")

	if !exists {
		if err := ensureRepoGet(ctx, rootDir, []string{repoURL}, noPrompt, theme, useColor); err != nil {
			return err
		}
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, filepath.Join(rootDir, "workspaces", workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}

	store, err := repo.Open(ctx, rootDir, repoURL, false)
	if err != nil {
		return err
	}
	fetchedRef, err := fetchPRRef(ctx, store.StorePath, req, workspaceID)
	if err != nil {
		return err
	}

	branch := workspaceID
	output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
	if _, err := workspace.AddWithBranch(ctx, rootDir, workspaceID, repoURL, "", branch, fetchedRef, false); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("review failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, repos)
	renderSuggestion(renderer, useColor, wsDir)
	return nil
}

type issueRequest struct {
	Provider string
	Host     string
	Owner    string
	Repo     string
	Number   int
}

func parseIssueURL(raw string) (issueRequest, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return issueRequest{}, fmt.Errorf("invalid issue URL: %w", err)
	}
	host := strings.TrimSpace(u.Hostname())
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return issueRequest{}, fmt.Errorf("invalid issue URL path: %s", u.Path)
	}

	for i := 0; i < len(parts)-1; i++ {
		if parts[i] != "issues" {
			continue
		}
		num, err := strconv.Atoi(parts[i+1])
		if err != nil {
			return issueRequest{}, fmt.Errorf("invalid issue number: %s", parts[i+1])
		}
		repoIdx := i - 1
		if repoIdx >= 1 && parts[repoIdx] == "-" {
			repoIdx--
		}
		if repoIdx < 1 {
			return issueRequest{}, fmt.Errorf("invalid issue URL path: %s", u.Path)
		}
		ownerParts := parts[:repoIdx]
		provider := issueProvider(host, repoIdx, i)
		if provider == "gitlab" {
			if len(ownerParts) != 1 {
				return issueRequest{}, fmt.Errorf("nested groups are not supported: %s", strings.Join(ownerParts, "/"))
			}
		} else if len(ownerParts) != 1 {
			return issueRequest{}, fmt.Errorf("invalid issue URL path: %s", u.Path)
		}
		return issueRequest{
			Provider: provider,
			Host:     host,
			Owner:    ownerParts[0],
			Repo:     parts[repoIdx],
			Number:   num,
		}, nil
	}

	return issueRequest{}, fmt.Errorf("unsupported issue URL: %s", raw)
}

func issueProvider(host string, repoIdx, issueIdx int) string {
	lowerHost := strings.ToLower(strings.TrimSpace(host))
	if repoIdx < issueIdx-1 || strings.Contains(lowerHost, "gitlab") {
		return "gitlab"
	}
	if strings.Contains(lowerHost, "bitbucket") {
		return "bitbucket"
	}
	return "github"
}

type prRequest struct {
	Provider string
	Host     string
	Owner    string
	Repo     string
	Number   int
}

func parsePRURL(raw string) (prRequest, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return prRequest{}, fmt.Errorf("invalid PR/MR URL: %w", err)
	}
	host := strings.TrimSpace(u.Hostname())
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return prRequest{}, fmt.Errorf("invalid PR/MR URL path: %s", u.Path)
	}

	// GitHub style: /owner/repo/pull/123
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "pull" && i >= 2 {
			num, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return prRequest{}, fmt.Errorf("invalid PR number: %s", parts[i+1])
			}
			return prRequest{
				Provider: "github",
				Host:     host,
				Owner:    parts[i-2],
				Repo:     parts[i-1],
				Number:   num,
			}, nil
		}
	}

	// Bitbucket Cloud style: /owner/repo/pull-requests/123
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "pull-requests" && i >= 2 {
			num, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return prRequest{}, fmt.Errorf("invalid PR number: %s", parts[i+1])
			}
			return prRequest{
				Provider: "bitbucket",
				Host:     host,
				Owner:    parts[i-2],
				Repo:     parts[i-1],
				Number:   num,
			}, nil
		}
	}

	// GitLab style: /group/repo/-/merge_requests/123
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "merge_requests" && i >= 2 {
			num, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return prRequest{}, fmt.Errorf("invalid MR number: %s", parts[i+1])
			}
			repoIdx := i - 1
			if repoIdx >= 1 && parts[repoIdx] == "-" {
				repoIdx--
			}
			if repoIdx < 1 {
				return prRequest{}, fmt.Errorf("invalid MR URL path: %s", u.Path)
			}
			ownerParts := parts[:repoIdx]
			if len(ownerParts) != 1 {
				return prRequest{}, fmt.Errorf("nested groups are not supported: %s", strings.Join(ownerParts, "/"))
			}
			return prRequest{
				Provider: "gitlab",
				Host:     host,
				Owner:    ownerParts[0],
				Repo:     parts[repoIdx],
				Number:   num,
			}, nil
		}
	}

	return prRequest{}, fmt.Errorf("unsupported PR/MR URL: %s", raw)
}

func buildRepoURL(req prRequest) string {
	return buildRepoURLFromParts(req.Host, req.Owner, req.Repo)
}

func buildRepoURLFromParts(host, owner, repo string) string {
	repoName := strings.TrimSuffix(repo, ".git")
	switch strings.ToLower(strings.TrimSpace(defaultRepoProtocol)) {
	case "https":
		return fmt.Sprintf("https://%s/%s/%s.git", host, owner, repoName)
	default:
		return fmt.Sprintf("git@%s:%s/%s.git", host, owner, repoName)
	}
}

func fetchPRRef(ctx context.Context, storePath string, req prRequest, workspaceID string) (string, error) {
	if strings.TrimSpace(storePath) == "" {
		return "", fmt.Errorf("store path is required")
	}
	if workspaceID == "" {
		return "", fmt.Errorf("workspace id is required")
	}
	var srcRef string
	switch strings.ToLower(req.Provider) {
	case "github":
		srcRef = fmt.Sprintf("pull/%d/head", req.Number)
	case "gitlab":
		srcRef = fmt.Sprintf("merge-requests/%d/head", req.Number)
	case "bitbucket":
		srcRef = fmt.Sprintf("refs/pull-requests/%d/from", req.Number)
	default:
		return "", fmt.Errorf("unsupported provider: %s", req.Provider)
	}
	destRef := fmt.Sprintf("refs/remotes/gws-review/%s", workspaceID)
	spec := fmt.Sprintf("%s:%s", srcRef, destRef)
	gitcmd.Logf("git fetch origin %s", spec)
	if _, err := gitcmd.Run(ctx, []string{"fetch", "origin", spec}, gitcmd.Options{Dir: storePath}); err != nil {
		return "", err
	}
	return destRef, nil
}

func promptTemplateAndID(rootDir, templateName, workspaceID string, theme ui.Theme, useColor bool) (string, string, error) {
	file, err := template.Load(rootDir)
	if err != nil {
		return "", "", err
	}
	names := template.Names(file)
	if len(names) == 0 {
		return "", "", fmt.Errorf("no templates found in %s", filepath.Join(rootDir, template.FileName))
	}
	templateName, workspaceID, err = ui.PromptNewWorkspaceInputs("gws new", names, templateName, workspaceID, theme, useColor)
	if err != nil {
		return "", "", err
	}
	return templateName, workspaceID, nil
}

func promptTemplateBranches(ctx context.Context, tmpl template.Template, workspaceID string, theme ui.Theme, useColor bool) ([]string, error) {
	if len(tmpl.Repos) == 0 {
		return nil, nil
	}
	branches := make([]string, len(tmpl.Repos))
	used := map[string]int{}

	for i, repoSpec := range tmpl.Repos {
		alias := displayRepoName(repoSpec)
		if strings.TrimSpace(alias) == "" {
			alias = fmt.Sprintf("repo #%d", i+1)
		}
		label := fmt.Sprintf("branch for %s", alias)

		for {
			value, err := ui.PromptInputInline(label, workspaceID, func(v string) error {
				return workspace.ValidateBranchName(ctx, v)
			}, theme, useColor)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(value) == "" {
				value = workspaceID
			}

			if prevIndex, exists := used[value]; exists {
				warnLabel := fmt.Sprintf("branch %q already used for repo #%d; use again?", value, prevIndex+1)
				confirm, err := ui.PromptConfirmInline(warnLabel, theme, useColor)
				if err != nil {
					return nil, err
				}
				if !confirm {
					continue
				}
			}

			branches[i] = value
			used[value] = i
			break
		}
	}

	return branches, nil
}

func renderWorkspaceRepos(r *ui.Renderer, repos []workspace.Repo, extraIndent string) {
	for i, repo := range repos {
		prefix := "├─ "
		if i == len(repos)-1 {
			prefix = "└─ "
		}
		name := formatRepoName(repo.Alias, repo.RepoKey)
		if r != nil {
			r.TreeLineBranchMuted(extraIndent+prefix, name, repo.Branch)
			continue
		}
		label := formatRepoLabel(name, repo.Branch)
		line := fmt.Sprintf("%s%s%s%s", output.Indent, extraIndent, prefix, label)
		fmt.Fprintln(os.Stdout, line)
	}
}

func renderWorkspaceBlock(r *ui.Renderer, workspaceID string, repos []workspace.Repo) {
	if r != nil {
		r.Bullet(fmt.Sprintf("%s (repos: %d)", workspaceID, len(repos)))
		renderWorkspaceRepos(r, repos, output.Indent)
		return
	}
	fmt.Fprintf(os.Stdout, "%s%s %s (repos: %d)\n", output.Indent, output.StepPrefix, workspaceID, len(repos))
	renderWorkspaceRepos(nil, repos, output.Indent)
}

func loadWorkspaceRepos(ctx context.Context, wsDir string) ([]workspace.Repo, []error, error) {
	repos, warnings, err := workspace.ScanRepos(ctx, wsDir)
	if err != nil {
		return nil, warnings, err
	}
	return repos, warnings, nil
}

func buildWorkspaceChoices(ctx context.Context, entries []workspace.Entry) []ui.WorkspaceChoice {
	var choices []ui.WorkspaceChoice
	for _, entry := range entries {
		choices = append(choices, buildWorkspaceChoice(ctx, entry))
	}
	return choices
}

func buildWorkspaceChoice(ctx context.Context, entry workspace.Entry) ui.WorkspaceChoice {
	choice := ui.WorkspaceChoice{ID: entry.WorkspaceID}
	repos, _, err := workspace.ScanRepos(ctx, entry.WorkspacePath)
	if err != nil {
		return choice
	}
	for _, repoEntry := range repos {
		name := formatRepoName(repoEntry.Alias, repoEntry.RepoKey)
		label := formatRepoLabel(name, repoEntry.Branch)
		choice.Repos = append(choice.Repos, ui.PromptChoice{
			Label: label,
			Value: displayRepoKey(repoEntry.RepoKey),
		})
	}
	return choice
}

func displayRepoKey(repoKey string) string {
	display := strings.TrimSuffix(repoKey, ".git")
	display = strings.TrimSuffix(display, "/")
	return display
}

func displayTemplateRepo(repoSpec string) string {
	trimmed := strings.TrimSpace(repoSpec)
	if trimmed == "" {
		return ""
	}
	spec, err := repospec.Normalize(trimmed)
	if err != nil {
		return trimmed
	}
	return fmt.Sprintf("git@%s:%s/%s.git", spec.Host, spec.Owner, spec.Repo)
}

func displayRepoSpec(repoSpec string) string {
	return displayTemplateRepo(repoSpec)
}

func displayRepoName(repoSpec string) string {
	trimmed := strings.TrimSpace(repoSpec)
	if trimmed == "" {
		return ""
	}
	spec, err := repospec.Normalize(trimmed)
	if err != nil || spec.Repo == "" {
		return trimmed
	}
	return spec.Repo
}

func repoDestForSpec(rootDir, repoSpec string) string {
	store := repoStoreRel(rootDir, repoSpec)
	src := repoSrcRel(rootDir, repoSpec)
	if store != "" && src != "" {
		return fmt.Sprintf("%s, %s", store, src)
	}
	if store != "" {
		return store
	}
	return src
}

func repoStoreRel(rootDir, repoSpec string) string {
	spec, err := repospec.Normalize(strings.TrimSpace(repoSpec))
	if err != nil {
		return ""
	}
	storePath := filepath.Join(rootDir, "bare", spec.Host, spec.Owner, spec.Repo+".git")
	return relPath(rootDir, storePath)
}

func repoSrcRel(rootDir, repoSpec string) string {
	spec, err := repospec.Normalize(strings.TrimSpace(repoSpec))
	if err != nil {
		return ""
	}
	srcPath := filepath.Join(rootDir, "src", spec.Host, spec.Owner, spec.Repo)
	return relPath(rootDir, srcPath)
}

func repoSrcAbs(rootDir, repoSpec string) string {
	spec, err := repospec.Normalize(strings.TrimSpace(repoSpec))
	if err != nil {
		return ""
	}
	return filepath.Join(rootDir, "src", spec.Host, spec.Owner, spec.Repo)
}

func worktreeDest(rootDir, workspaceID, repoSpec string) string {
	spec, err := repospec.Normalize(strings.TrimSpace(repoSpec))
	if err != nil || spec.Repo == "" {
		return ""
	}
	wsPath := filepath.Join(rootDir, "workspaces", workspaceID, spec.Repo)
	return relPath(rootDir, wsPath)
}

func relPath(rootDir, path string) string {
	if strings.TrimSpace(rootDir) == "" || strings.TrimSpace(path) == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func formatStep(action, target, dest string) string {
	parts := []string{strings.TrimSpace(action)}
	if strings.TrimSpace(target) != "" {
		parts = append(parts, truncateMiddle(strings.TrimSpace(target), 80))
	}
	text := strings.Join(parts, " ")
	if strings.TrimSpace(dest) != "" {
		return fmt.Sprintf("%s -> %s", text, truncateMiddle(dest, 80))
	}
	return text
}

func formatStepWithIndex(action, target, dest string, index, total int) string {
	if total > 0 {
		if strings.TrimSpace(target) != "" {
			target = fmt.Sprintf("%s (%d/%d)", target, index, total)
		} else {
			action = fmt.Sprintf("%s (%d/%d)", action, index, total)
		}
	}
	return formatStep(action, target, dest)
}

func truncateMiddle(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if max <= 0 || len(trimmed) <= max {
		return trimmed
	}
	if max < 10 {
		return trimmed[:max]
	}
	keep := (max - 3) / 2
	return fmt.Sprintf("%s...%s", trimmed[:keep], trimmed[len(trimmed)-keep:])
}

func startSteps(renderer *ui.Renderer) {
	if renderer == nil {
		return
	}
	renderer.Section("Steps")
}

func ensureRepoGet(ctx context.Context, rootDir string, repoSpecs []string, noPrompt bool, theme ui.Theme, useColor bool) error {
	if len(repoSpecs) == 0 {
		return nil
	}
	var missing []string
	for _, repoSpec := range repoSpecs {
		if strings.TrimSpace(repoSpec) == "" {
			continue
		}
		missing = append(missing, repoSpec)
	}
	if len(missing) == 0 {
		return nil
	}
	if noPrompt {
		return fmt.Errorf("repo get required for: %s", strings.Join(missing, ", "))
	}
	label := "repos"
	if len(missing) == 1 {
		label = "repo"
	}
	output.Step(fmt.Sprintf("repo get required for %d %s", len(missing), label))
	for _, repoSpec := range missing {
		output.Log(fmt.Sprintf("gws repo get %s", displayRepoSpec(repoSpec)))
	}
	confirm, err := ui.PromptConfirmInline("run now?", theme, useColor)
	if err != nil {
		return err
	}
	if !confirm {
		return fmt.Errorf("repo get required for: %s", strings.Join(missing, ", "))
	}
	for i, repoSpec := range missing {
		output.Step(formatStepWithIndex("repo get", displayRepoSpec(repoSpec), repoDestForSpec(rootDir, repoSpec), i+1, len(missing)))
		if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
			return err
		}
	}
	return nil
}

func renderSuggestion(r *ui.Renderer, useColor bool, path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	renderSuggestions(r, useColor, []string{fmt.Sprintf("cd %s", path)})
}

func renderSuggestions(r *ui.Renderer, useColor bool, lines []string) {
	if !useColor || r == nil {
		return
	}
	var filtered []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		return
	}
	r.Blank()
	r.Section("Suggestion")
	for _, line := range filtered {
		r.Bullet(line)
	}
}

func repoSpecFromKey(repoKey string) string {
	trimmed := strings.TrimSuffix(repoKey, ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return repoKey
	}
	host := parts[0]
	owner := parts[1]
	repo := parts[2]
	if strings.EqualFold(strings.TrimSpace(defaultRepoProtocol), "ssh") {
		return fmt.Sprintf("git@%s:%s/%s.git", host, owner, repo)
	}
	return fmt.Sprintf("https://%s/%s/%s.git", host, owner, repo)
}

func printRepoGetCommands(repos []string) {
	fmt.Fprintf(os.Stdout, "%scommands:\n", output.Indent)
	for _, repoSpec := range repos {
		fmt.Fprintf(os.Stdout, "%sgws repo get %s\n", output.Indent+output.Indent, repoSpec)
	}
}

type statusDetail struct {
	text string
	warn bool
}

func buildStatusDetails(repo workspace.RepoStatus) []statusDetail {
	var details []statusDetail
	head := strings.TrimSpace(repo.Head)
	if head != "" {
		details = append(details, statusDetail{text: fmt.Sprintf("head: %s", head)})
	}
	if repo.StagedCount == 0 && repo.UnstagedCount == 0 && repo.UntrackedCount == 0 && repo.UnmergedCount == 0 {
		details = append(details, statusDetail{text: "changes: clean"})
		return details
	}
	if repo.StagedCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("staged: %d", repo.StagedCount), warn: true})
	}
	if repo.UnstagedCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("unstaged: %d", repo.UnstagedCount), warn: true})
	}
	if repo.UntrackedCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("untracked: %d", repo.UntrackedCount), warn: true})
	}
	if repo.UnmergedCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("unmerged: %d", repo.UnmergedCount), warn: true})
	}
	if repo.AheadCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("ahead: %d", repo.AheadCount), warn: true})
	}
	if repo.BehindCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("behind: %d", repo.BehindCount), warn: true})
	}
	return details
}

func collectRemoveWarnings(ctx context.Context, rootDir, workspaceID string) []string {
	status, err := workspace.Status(ctx, rootDir, workspaceID)
	if err != nil {
		return []string{fmt.Sprintf("status check failed: %s", compactError(err))}
	}
	return buildRemoveWarnings(status)
}

func buildRemoveWarnings(status workspace.StatusResult) []string {
	var warnings []string
	for _, warning := range status.Warnings {
		warnings = append(warnings, compactError(warning))
	}
	for _, repo := range status.Repos {
		name := strings.TrimSpace(repo.Alias)
		if name == "" && strings.TrimSpace(repo.WorktreePath) != "" {
			name = filepath.Base(repo.WorktreePath)
		}
		if name == "" {
			name = "repo"
		}
		if repo.Error != nil {
			warnings = append(warnings, fmt.Sprintf("%s: status error (%s)", name, compactError(repo.Error)))
			continue
		}
		if repo.AheadCount > 0 {
			upstream := repo.Upstream
			if strings.TrimSpace(upstream) == "" {
				upstream = "upstream"
			}
			warnings = append(warnings, fmt.Sprintf("%s: ahead of %s by %d", name, upstream, repo.AheadCount))
		}
		if strings.TrimSpace(repo.Upstream) == "" {
			warnings = append(warnings, fmt.Sprintf("%s: upstream not set", name))
		}
	}
	return warnings
}

func issueDetails(issue doctor.Issue) []string {
	var details []string
	if strings.TrimSpace(issue.Path) != "" {
		details = append(details, fmt.Sprintf("path: %s", issue.Path))
	}
	if strings.TrimSpace(issue.Message) != "" {
		details = append(details, fmt.Sprintf("message: %s", issue.Message))
	}
	return details
}

type treeLineStyle int

const (
	treeLineNormal treeLineStyle = iota
	treeLineWarn
	treeLineError
)

func renderTreeLines(r *ui.Renderer, lines []string, style treeLineStyle) {
	for i, line := range lines {
		prefix := "├─ "
		if i == len(lines)-1 {
			prefix = "└─ "
		}
		switch style {
		case treeLineWarn:
			r.TreeLineWarn(output.Indent+prefix, line)
		case treeLineError:
			r.TreeLineError(output.Indent+prefix, line)
		default:
			r.TreeLine(output.Indent+prefix, line)
		}
	}
}

func renderWarningsSection(r *ui.Renderer, title string, warnings []string, leadBlank bool) {
	if r == nil || len(warnings) == 0 {
		return
	}
	if leadBlank {
		r.Blank()
	}
	r.Section("Info")
	r.Bullet(title)
	renderTreeLines(r, warnings, treeLineWarn)
}

func formatRepoName(alias, repoKey string) string {
	name := strings.TrimSpace(alias)
	if name != "" {
		return name
	}
	return repoKey
}

func formatRepoLabel(name, branch string) string {
	if strings.TrimSpace(branch) != "" {
		return fmt.Sprintf("%s (branch: %s)", name, branch)
	}
	return name
}

func appendWarningLines(lines []string, prefix string, warnings []error) []string {
	for _, warning := range warnings {
		message := compactError(warning)
		if strings.TrimSpace(prefix) != "" {
			message = fmt.Sprintf("%s: %s", prefix, message)
		}
		lines = append(lines, message)
	}
	return lines
}

func classifyWorkspaceRemoval(ctx context.Context, rootDir string, entries []workspace.Entry) ([]ui.WorkspaceChoice, []ui.BlockedChoice) {
	var removable []ui.WorkspaceChoice
	var blocked []ui.BlockedChoice
	for _, entry := range entries {
		reason := workspaceRemoveReason(ctx, rootDir, entry)
		if strings.TrimSpace(reason) == "" {
			removable = append(removable, buildWorkspaceChoice(ctx, entry))
			continue
		}
		blocked = append(blocked, ui.BlockedChoice{
			Label: fmt.Sprintf("%s (%s)", entry.WorkspaceID, reason),
		})
	}
	return removable, blocked
}

func workspaceRemoveReason(ctx context.Context, rootDir string, entry workspace.Entry) string {
	status, err := workspace.Status(ctx, rootDir, entry.WorkspaceID)
	if err != nil {
		return fmt.Sprintf("status: %s", compactError(err))
	}
	return buildWorkspaceRemoveReason(status)
}

func buildWorkspaceRemoveReason(status workspace.StatusResult) string {
	var dirtyRepos []string
	var errorRepos []string
	for _, repo := range status.Repos {
		name := strings.TrimSpace(repo.Alias)
		if name == "" {
			name = "unknown"
		}
		if repo.Error != nil {
			errorRepos = append(errorRepos, fmt.Sprintf("%s (%s)", name, compactError(repo.Error)))
			continue
		}
		if repo.Dirty {
			detail := formatDirtySummary(repo)
			if detail == "" {
				detail = "dirty"
			}
			dirtyRepos = append(dirtyRepos, fmt.Sprintf("%s (%s)", name, detail))
		}
	}
	var reasons []string
	if len(errorRepos) > 0 {
		reasons = append(reasons, fmt.Sprintf("status error: %s", strings.Join(errorRepos, ", ")))
	}
	if len(dirtyRepos) > 0 {
		reasons = append(reasons, fmt.Sprintf("dirty: %s", strings.Join(dirtyRepos, ", ")))
	}
	return strings.Join(reasons, "; ")
}

func formatDirtySummary(repo workspace.RepoStatus) string {
	var parts []string
	if repo.StagedCount > 0 {
		parts = append(parts, fmt.Sprintf("staged=%d", repo.StagedCount))
	}
	if repo.UnstagedCount > 0 {
		parts = append(parts, fmt.Sprintf("unstaged=%d", repo.UnstagedCount))
	}
	if repo.UntrackedCount > 0 {
		parts = append(parts, fmt.Sprintf("untracked=%d", repo.UntrackedCount))
	}
	if repo.UnmergedCount > 0 {
		parts = append(parts, fmt.Sprintf("unmerged=%d", repo.UnmergedCount))
	}
	return strings.Join(parts, ", ")
}

func compactError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "unknown error"
	}
	return strings.Join(strings.Fields(msg), " ")
}

func preflightTemplateRepos(ctx context.Context, rootDir string, tmpl template.Template) ([]string, error) {
	var missing []string
	for _, repoSpec := range tmpl.Repos {
		if strings.TrimSpace(repoSpec) == "" {
			return nil, fmt.Errorf("template repo is empty")
		}
		_, exists, err := repo.Exists(rootDir, repoSpec)
		if err != nil {
			return nil, err
		}
		if !exists {
			missing = append(missing, repoSpec)
		}
	}
	return missing, nil
}

func applyTemplate(ctx context.Context, rootDir, workspaceID string, tmpl template.Template, branches []string) error {
	total := len(tmpl.Repos)
	for i, repoSpec := range tmpl.Repos {
		branch := workspaceID
		if len(branches) == len(tmpl.Repos) && i < len(branches) && strings.TrimSpace(branches[i]) != "" {
			branch = branches[i]
		}
		output.Step(formatStepWithIndex("worktree add", displayRepoName(repoSpec), worktreeDest(rootDir, workspaceID, repoSpec), i+1, total))
		if _, err := workspace.AddWithBranch(ctx, rootDir, workspaceID, repoSpec, "", branch, "", true); err != nil {
			return err
		}
	}
	return nil
}

func runWorkspaceList(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printLsHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gws ls")
	}
	entries, warnings, err := workspace.List(rootDir)
	if err != nil {
		return err
	}
	writeWorkspaceListText(ctx, entries, warnings)
	return nil
}

func runWorkspaceStatus(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printStatusHelp(os.Stdout)
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: gws status [<WORKSPACE_ID>]")
	}
	workspaceID := ""
	if len(args) == 1 {
		workspaceID = args[0]
	}
	if workspaceID == "" {
		workspaces, wsWarn, err := workspace.List(rootDir)
		if err != nil {
			return err
		}
		if len(wsWarn) > 0 {
			// ignore warnings for selection
		}
		workspaceChoices := buildWorkspaceChoices(ctx, workspaces)
		if len(workspaceChoices) == 0 {
			return fmt.Errorf("no workspaces found")
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		workspaceID, err = ui.PromptWorkspace("gws status", workspaceChoices, theme, useColor)
		if err != nil {
			return err
		}
	}
	result, err := workspace.Status(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}

	writeWorkspaceStatusText(result)
	return nil
}

func runWorkspaceRemove(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printRmHelp(os.Stdout)
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: gws rm [<WORKSPACE_ID>]")
	}
	workspaceID := ""
	if len(args) == 1 {
		workspaceID = args[0]
	}

	if workspaceID == "" {
		workspaces, wsWarn, err := workspace.List(rootDir)
		if err != nil {
			return err
		}
		if len(wsWarn) > 0 {
			// ignore warnings for selection
		}
		if len(workspaces) == 0 {
			return fmt.Errorf("no workspaces found")
		}
		removable, blocked := classifyWorkspaceRemoval(ctx, rootDir, workspaces)
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		if len(removable) == 0 {
			renderer := ui.NewRenderer(os.Stdout, theme, useColor)
			renderer.Section("Info")
			renderer.Bullet("no removable workspaces")
			if len(blocked) > 0 {
				renderer.Bullet("blocked workspaces")
				for _, item := range blocked {
					renderer.TreeLineWarn(output.LogConnector+" ", item.Label)
				}
			}
			return fmt.Errorf("no removable workspaces")
		}
		workspaceID, err = ui.PromptWorkspaceWithBlocked("gws rm", removable, blocked, theme, useColor)
		if err != nil {
			return err
		}
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	removeWarnings := collectRemoveWarnings(ctx, rootDir, workspaceID)
	if len(removeWarnings) > 0 {
		renderWarningsSection(renderer, "possible unpushed commits", removeWarnings, false)
		renderer.Blank()
	}
	renderer.Section("Steps")
	output.Step(formatStep("remove workspace", workspaceID, relPath(rootDir, filepath.Join(rootDir, "workspaces", workspaceID))))

	if err := workspace.Remove(ctx, rootDir, workspaceID); err != nil {
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	renderer.Bullet(fmt.Sprintf("%s removed", workspaceID))
	return nil
}

func writeWorkspaceStatusText(result workspace.StatusResult) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Result")

	for _, repo := range result.Repos {
		label := repo.Alias
		if strings.TrimSpace(label) == "" {
			label = filepath.Base(repo.WorktreePath)
		}
		if strings.TrimSpace(repo.Branch) != "" {
			label = fmt.Sprintf("%s (branch: %s)", label, repo.Branch)
		}
		renderer.Bullet(label)

		details := buildStatusDetails(repo)
		for i, detail := range details {
			prefix := "├─ "
			if i == len(details)-1 {
				prefix = "└─ "
			}
			prefix = output.Indent + prefix
			if detail.warn {
				renderer.TreeLineWarn(prefix, detail.text)
			} else {
				renderer.TreeLineBranchMuted(prefix, detail.text, "")
			}
		}
		if repo.Error != nil {
			renderer.Warn(fmt.Sprintf("warning: %s: %v", repo.Alias, repo.Error))
		}
	}
	warningLines := appendWarningLines(nil, "", result.Warnings)
	renderWarningsSection(renderer, "warnings", warningLines, true)
}

func writeWorkspaceListText(ctx context.Context, entries []workspace.Entry, warnings []error) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Result")
	var repoWarnings []string
	for _, entry := range entries {
		repos, warnings, err := workspace.ScanRepos(ctx, entry.WorkspacePath)
		if err != nil {
			repoWarnings = append(repoWarnings, fmt.Sprintf("%s: %s", entry.WorkspaceID, compactError(err)))
		}
		repoWarnings = appendWarningLines(repoWarnings, entry.WorkspaceID, warnings)
		renderWorkspaceBlock(renderer, entry.WorkspaceID, repos)
	}
	repoWarnings = appendWarningLines(repoWarnings, "", warnings)
	renderWarningsSection(renderer, "warnings", repoWarnings, true)
}

func writeRepoListText(entries []repo.Entry, warnings []error) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Result")
	for _, entry := range entries {
		renderer.Result(fmt.Sprintf("%s\t%s", entry.RepoKey, entry.StorePath))
	}
	warningLines := appendWarningLines(nil, "", warnings)
	renderWarningsSection(renderer, "warnings", warningLines, true)
}

func writeTemplateListText(file template.File, names []string) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Result")
	if len(names) == 0 {
		renderer.Bullet("no templates found")
		return
	}
	for _, name := range names {
		renderer.Bullet(name)
		tmpl, ok := file.Templates[name]
		if !ok {
			continue
		}
		var repos []string
		for _, repoSpec := range tmpl.Repos {
			display := displayTemplateRepo(repoSpec)
			if strings.TrimSpace(display) == "" {
				continue
			}
			repos = append(repos, display)
		}
		renderTreeLines(renderer, repos, treeLineNormal)
	}
}

func writeTemplateShowText(name string, tmpl template.Template) {
	fmt.Fprintf(os.Stdout, "%s\n", name)
	for _, repo := range tmpl.Repos {
		fmt.Fprintf(os.Stdout, " - %s\n", repo)
	}
}

func writeInitText(result initcmd.Result) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Steps")
	if len(result.CreatedDirs) == 0 && len(result.CreatedFiles) == 0 {
		renderer.Bullet("no changes")
	} else {
		for _, dir := range result.CreatedDirs {
			renderer.Bullet(fmt.Sprintf("create dir %s", dir))
		}
		for _, file := range result.CreatedFiles {
			renderer.Bullet(fmt.Sprintf("create file %s", file))
		}
	}

	renderer.Blank()
	renderer.Section("Result")
	renderer.Result(fmt.Sprintf("root: %s", result.RootDir))

	var skipped []string
	for _, dir := range result.SkippedDirs {
		skipped = append(skipped, fmt.Sprintf("dir: %s", dir))
	}
	for _, file := range result.SkippedFiles {
		skipped = append(skipped, fmt.Sprintf("file: %s", file))
	}
	if len(skipped) > 0 {
		renderer.Blank()
		renderer.Section("Info")
		renderer.Bullet("already exists")
		renderTreeLines(renderer, skipped, treeLineNormal)
	}

	renderSuggestions(renderer, useColor, []string{
		fmt.Sprintf("Edit templates.yaml: %s", filepath.Join(result.RootDir, "templates.yaml")),
	})
}
func writeDoctorText(result doctor.Result, fixed []string) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Result")

	if len(result.Issues) == 0 {
		renderer.Bullet("no issues found")
	} else {
		for _, issue := range result.Issues {
			renderer.BulletError(issue.Kind)
			details := issueDetails(issue)
			renderTreeLines(renderer, details, treeLineError)
		}
	}

	if len(fixed) > 0 {
		renderer.Bullet(fmt.Sprintf("fixed (%d)", len(fixed)))
		var lines []string
		for _, path := range fixed {
			lines = append(lines, path)
		}
		renderTreeLines(renderer, lines, treeLineNormal)
	}

	if len(result.Warnings) > 0 {
		renderer.Blank()
		renderer.Section("Info")
		renderer.Bullet("warnings")
		var lines []string
		for _, warning := range result.Warnings {
			lines = append(lines, warning.Error())
		}
		renderTreeLines(renderer, lines, treeLineWarn)
	}
}

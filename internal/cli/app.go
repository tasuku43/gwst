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
	case "create":
		return runCreate(ctx, rootDir, args[1:], noPrompt)
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
	case "path":
		return runPath(rootDir, args[1:], noPrompt)
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
	case "add":
		return runTemplateAdd(ctx, rootDir, args[1:], noPrompt)
	case "rm":
		return runTemplateRemove(ctx, rootDir, args[1:], noPrompt)
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

type stringFlag struct {
	value string
	set   bool
}

func (s *stringFlag) String() string {
	return s.value
}

func (s *stringFlag) Set(value string) error {
	s.value = value
	s.set = true
	return nil
}

type boolFlag struct {
	value bool
	set   bool
}

func (b *boolFlag) String() string {
	if b == nil {
		return "false"
	}
	if b.value {
		return "true"
	}
	return "false"
}

func (b *boolFlag) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	b.value = parsed
	b.set = true
	return nil
}

func (b *boolFlag) IsBoolFlag() bool {
	return true
}

func uniqueStringsPreserve(items []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func runTemplateAdd(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	addFlags := flag.NewFlagSet("template add", flag.ContinueOnError)
	var helpFlag bool
	var repos stringSliceFlag
	addFlags.Var(&repos, "repo", "repo spec (repeatable)")
	addFlags.BoolVar(&helpFlag, "help", false, "show help")
	addFlags.BoolVar(&helpFlag, "h", false, "show help")
	addFlags.SetOutput(os.Stdout)
	addFlags.Usage = func() {
		printTemplateAddHelp(os.Stdout)
	}
	if err := addFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printTemplateAddHelp(os.Stdout)
		return nil
	}
	if addFlags.NArg() > 1 {
		return fmt.Errorf("usage: gws template add [<name>] [--repo <repo> ...]")
	}

	name := ""
	if addFlags.NArg() == 1 {
		name = addFlags.Arg(0)
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
		name, repoSpecs, err = ui.PromptTemplateRepos("gws template add", name, choices, theme, useColor)
		if err != nil {
			return err
		}
		repoSpecs = template.NormalizeRepos(repoSpecs)
	} else {
		if strings.TrimSpace(name) == "" {
			if noPrompt {
				return fmt.Errorf("template name is required with --no-prompt")
			}
			name, err = ui.PromptTemplateName("gws template add", "", theme, useColor)
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
			name, selected, err = ui.PromptTemplateRepos("gws template add", name, choices, theme, useColor)
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
		if _, _, err := repo.Normalize(repoSpec); err != nil {
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

func runTemplateRemove(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	rmFlags := flag.NewFlagSet("template rm", flag.ContinueOnError)
	var helpFlag bool
	rmFlags.BoolVar(&helpFlag, "help", false, "show help")
	rmFlags.BoolVar(&helpFlag, "h", false, "show help")
	rmFlags.SetOutput(os.Stdout)
	rmFlags.Usage = func() {
		printTemplateRmHelp(os.Stdout)
	}
	if err := rmFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printTemplateRmHelp(os.Stdout)
		return nil
	}

	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}

	var names []string
	showInputs := true
	if rmFlags.NArg() > 0 {
		names = uniqueStringsPreserve(rmFlags.Args())
		for _, name := range names {
			if err := template.ValidateName(name); err != nil {
				return err
			}
		}
	} else {
		showInputs = false
		if noPrompt {
			return fmt.Errorf("template name is required with --no-prompt")
		}
		templates := template.Names(file)
		if len(templates) == 0 {
			return fmt.Errorf("no templates found in %s", filepath.Join(rootDir, template.FileName))
		}
		var choices []ui.PromptChoice
		for _, name := range templates {
			choices = append(choices, ui.PromptChoice{Label: name, Value: name})
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		selected, err := ui.PromptMultiSelect("gws template rm", "template", choices, theme, useColor)
		if err != nil {
			return err
		}
		names = uniqueStringsPreserve(selected)
	}

	for _, name := range names {
		if _, exists := file.Templates[name]; !exists {
			return fmt.Errorf("template not found: %s", name)
		}
	}

	for _, name := range names {
		delete(file.Templates, name)
	}

	if err := template.Save(rootDir, file); err != nil {
		return err
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)
	if showInputs {
		renderer.Section("Inputs")
		renderer.Bullet("templates")
		renderTreeLines(renderer, names, treeLineNormal)
		renderer.Blank()
	}
	renderer.Section("Steps")
	for i, name := range names {
		output.Step(formatStepWithIndex("remove template", name, relPath(rootDir, filepath.Join(rootDir, template.FileName)), i+1, len(names)))
	}
	renderer.Blank()
	renderer.Section("Result")
	for _, name := range names {
		renderer.Bullet(fmt.Sprintf("%s removed", name))
	}
	return nil
}

func runCreate(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	createFlags := flag.NewFlagSet("create", flag.ContinueOnError)
	var templateName stringFlag
	var reviewFlag boolFlag
	var issueFlag boolFlag
	var repoFlag stringFlag
	var workspaceID string
	var branch string
	var baseRef string
	var helpFlag bool
	createFlags.Var(&templateName, "template", "template name")
	createFlags.Var(&reviewFlag, "review", "create review workspace from PR")
	createFlags.Var(&issueFlag, "issue", "create issue workspace from issue")
	createFlags.Var(&repoFlag, "repo", "create workspace from repos")
	createFlags.StringVar(&workspaceID, "workspace-id", "", "workspace id")
	createFlags.StringVar(&branch, "branch", "", "branch name")
	createFlags.StringVar(&baseRef, "base", "", "base ref")
	createFlags.BoolVar(&helpFlag, "help", false, "show help")
	createFlags.BoolVar(&helpFlag, "h", false, "show help")
	createFlags.SetOutput(os.Stdout)
	createFlags.Usage = func() {
		printCreateHelp(os.Stdout)
	}
	if err := createFlags.Parse(normalizeCreateArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printCreateHelp(os.Stdout)
		return nil
	}

	workspaceID = strings.TrimSpace(workspaceID)
	branch = strings.TrimSpace(branch)
	baseRef = strings.TrimSpace(baseRef)
	templateName.value = strings.TrimSpace(templateName.value)

	templateMode := templateName.set
	reviewMode := reviewFlag.value
	issueMode := issueFlag.value
	repoMode := repoFlag.set
	modeCount := 0
	if templateMode {
		modeCount++
	}
	if reviewMode {
		modeCount++
	}
	if issueMode {
		modeCount++
	}
	if repoMode {
		modeCount++
	}
	if modeCount > 1 {
		return fmt.Errorf("specify exactly one mode: --template, --review, --issue, or --repo")
	}
	if modeCount == 0 {
		if noPrompt {
			return fmt.Errorf("mode is required when --no-prompt is set")
		}
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("interactive mode picker requires a TTY")
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		templateNames, tmplErr := loadTemplateNames(rootDir)
		repoChoices, repoErr := buildTemplateRepoChoices(rootDir)
		reviewChoices, reviewErr := buildReviewRepoChoices(rootDir)
		issueChoices, issueErr := buildIssueRepoChoices(rootDir)
		reviewPrompt, reviewByValue := toPromptChoices(reviewChoices)
		issuePrompt, issueByValue := toIssuePromptChoices(issueChoices)
		loadReview := func(value string) ([]ui.PromptChoice, error) {
			if reviewErr != nil {
				return nil, reviewErr
			}
			selected, ok := reviewByValue[value]
			if !ok {
				return nil, fmt.Errorf("selected repo not found")
			}
			prs, err := fetchGitHubPRs(ctx, selected.Host, selected.Owner, selected.Repo)
			if err != nil {
				return nil, err
			}
			return buildPRChoices(prs), nil
		}
		loadIssue := func(value string) ([]ui.PromptChoice, error) {
			if issueErr != nil {
				return nil, issueErr
			}
			selected, ok := issueByValue[value]
			if !ok {
				return nil, fmt.Errorf("selected repo not found")
			}
			if strings.ToLower(selected.Provider) != "github" {
				return nil, fmt.Errorf("issue picker supports GitHub only for now: %s", selected.Host)
			}
			issues, err := fetchGitHubIssues(ctx, selected.Host, selected.Owner, selected.Repo)
			if err != nil {
				return nil, err
			}
			return buildIssueChoices(issues), nil
		}
		loadTemplateRepos := func(name string) ([]string, error) {
			file, err := template.Load(rootDir)
			if err != nil {
				return nil, err
			}
			tmpl, ok := file.Templates[name]
			if !ok {
				return nil, fmt.Errorf("template not found: %s", name)
			}
			return append([]string(nil), tmpl.Repos...), nil
		}
		validateBranch := func(v string) error {
			return workspace.ValidateBranchName(ctx, v)
		}
		mode, tmplName, tmplWorkspaceID, tmplDesc, tmplBranches, reviewRepo, reviewPRs, issueRepo, issueSelections, repoSelected, err := ui.PromptCreateFlow("gws create", "", "", templateNames, tmplErr, repoChoices, repoErr, reviewPrompt, issuePrompt, loadReview, loadIssue, loadTemplateRepos, validateBranch, theme, useColor, "")
		if err != nil {
			return err
		}
		switch mode {
		case "template":
			inputs := createTemplateInputs{
				templateName: tmplName,
				workspaceID:  tmplWorkspaceID,
				description:  tmplDesc,
				branches:     tmplBranches,
				fromFlow:     true,
			}
			return runCreateTemplateWithInputs(ctx, rootDir, inputs, noPrompt)
		case "review":
			if err := runCreateReviewSelected(ctx, rootDir, noPrompt, reviewRepo, reviewPRs); err != nil {
				return err
			}
			return nil
		case "issue":
			if err := runCreateIssueSelected(ctx, rootDir, noPrompt, issueRepo, issueSelections); err != nil {
				return err
			}
			return nil
		case "repo":
			inputs := createRepoInputs{
				repos:       []string{repoSelected},
				workspaceID: tmplWorkspaceID,
				description: tmplDesc,
				branches:    tmplBranches,
				fromFlow:    true,
			}
			return runCreateRepoWithInputs(ctx, rootDir, inputs, noPrompt)
		default:
			return fmt.Errorf("unknown mode: %s", mode)
		}
	}

	remaining := createFlags.Args()
	if templateMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gws create --template <name> [<WORKSPACE_ID>]")
		}
		if len(remaining) == 1 {
			if workspaceID != "" && workspaceID != remaining[0] {
				return fmt.Errorf("workspace id is specified twice: %s and %s", workspaceID, remaining[0])
			}
			workspaceID = remaining[0]
		}
		return runCreateTemplate(ctx, rootDir, templateName.value, workspaceID, noPrompt)
	}
	if reviewMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gws create --review [<PR URL>]")
		}
		if workspaceID != "" || branch != "" || baseRef != "" {
			return fmt.Errorf("--workspace-id, --branch, and --base are not valid with --review")
		}
		prURL := ""
		if len(remaining) == 1 {
			prURL = remaining[0]
		}
		return runCreateReview(ctx, rootDir, prURL, noPrompt)
	}
	if issueMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gws create --issue [<ISSUE_URL>] [--workspace-id <id>] [--branch <name>] [--base <ref>]")
		}
		issueURL := ""
		if len(remaining) == 1 {
			issueURL = remaining[0]
		}
		return runCreateIssue(ctx, rootDir, issueURL, workspaceID, branch, baseRef, noPrompt)
	}
	if repoMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gws create --repo [<repo>]")
		}
		if branch != "" || baseRef != "" {
			return fmt.Errorf("--branch and --base are not valid with --repo")
		}
		repoSpec := strings.TrimSpace(repoFlag.value)
		if len(remaining) == 1 {
			if repoSpec != "" && repoSpec != remaining[0] {
				return fmt.Errorf("repo is specified twice: %s and %s", repoSpec, remaining[0])
			}
			repoSpec = remaining[0]
		}
		if repoSpec == "" {
			if noPrompt {
				return fmt.Errorf("--repo requires prompts or a repo argument")
			}
			if !isatty.IsTerminal(os.Stdin.Fd()) {
				return fmt.Errorf("repo selection requires a TTY")
			}
			theme := ui.DefaultTheme()
			useColor := isatty.IsTerminal(os.Stdout.Fd())
			repoChoices, repoErr := buildTemplateRepoChoices(rootDir)
			mode, _, tmplWorkspaceID, tmplDesc, tmplBranches, _, _, _, _, repoSelected, err := ui.PromptCreateFlow("gws create", "repo", workspaceID, nil, nil, repoChoices, repoErr, nil, nil, nil, nil, nil, func(v string) error {
				return workspace.ValidateBranchName(ctx, v)
			}, theme, useColor, "")
			if err != nil {
				return err
			}
			if mode != "repo" {
				return fmt.Errorf("unknown mode: %s", mode)
			}
			inputs := createRepoInputs{
				repos:       []string{repoSelected},
				workspaceID: tmplWorkspaceID,
				description: tmplDesc,
				branches:    tmplBranches,
				fromFlow:    true,
			}
			return runCreateRepoWithInputs(ctx, rootDir, inputs, noPrompt)
		}
		if !noPrompt {
			if !isatty.IsTerminal(os.Stdin.Fd()) {
				return fmt.Errorf("repo prompts require a TTY")
			}
			theme := ui.DefaultTheme()
			useColor := isatty.IsTerminal(os.Stdout.Fd())
			mode, _, tmplWorkspaceID, tmplDesc, tmplBranches, _, _, _, _, repoSelected, err := ui.PromptCreateFlow("gws create", "repo", workspaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, func(v string) error {
				return workspace.ValidateBranchName(ctx, v)
			}, theme, useColor, repoSpec)
			if err != nil {
				return err
			}
			if mode != "repo" {
				return fmt.Errorf("unknown mode: %s", mode)
			}
			if repoSelected != "" {
				repoSpec = repoSelected
			}
			inputs := createRepoInputs{
				repos:       []string{repoSpec},
				workspaceID: tmplWorkspaceID,
				description: tmplDesc,
				branches:    tmplBranches,
				fromFlow:    true,
			}
			return runCreateRepoWithInputs(ctx, rootDir, inputs, noPrompt)
		}
		inputs := createRepoInputs{
			repos:       []string{repoSpec},
			workspaceID: workspaceID,
			fromFlow:    false,
		}
		return runCreateRepoWithInputs(ctx, rootDir, inputs, noPrompt)
	}
	return fmt.Errorf("mode is required")
}

func normalizeCreateArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--template" || arg == "-template" {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				out = append(out, arg+"=")
				continue
			}
		}
		if arg == "--repo" || arg == "-repo" {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				out = append(out, arg+"=")
				continue
			}
		}
		out = append(out, arg)
	}
	return out
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
	var self bool
	var helpFlag bool
	doctorFlags.SetOutput(os.Stdout)
	doctorFlags.Usage = func() {
		printDoctorHelp(os.Stdout)
	}
	doctorFlags.BoolVar(&fix, "fix", false, "remove stale locks only")
	doctorFlags.BoolVar(&self, "self", false, "run self-diagnostics")
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
	if fix && self {
		return fmt.Errorf("usage: gws doctor [--fix | --self]")
	}
	if doctorFlags.NArg() != 0 {
		return fmt.Errorf("usage: gws doctor [--fix | --self]")
	}
	now := time.Now().UTC()
	if self {
		result, err := doctor.SelfCheck(ctx)
		if err != nil {
			return err
		}
		writeDoctorSelfText(result)
		return nil
	}
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
	renderer.Bullet(fmt.Sprintf("%s %s", store.RepoKey, store.StorePath))
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
		printCreateHelp(os.Stdout)
	}
	if err := newFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printCreateHelp(os.Stdout)
		return nil
	}
	if newFlags.NArg() > 1 {
		return fmt.Errorf("usage: gws create --template <name> [<WORKSPACE_ID>]")
	}

	workspaceID := ""
	if newFlags.NArg() == 1 {
		workspaceID = newFlags.Arg(0)
	}

	return runCreateTemplate(ctx, rootDir, templateName, workspaceID, noPrompt)
}

type createTemplateInputs struct {
	templateName string
	workspaceID  string
	description  string
	branches     []string
	fromFlow     bool
}

type createRepoInputs struct {
	repos       []string
	workspaceID string
	description string
	branches    []string
	fromFlow    bool
}

func runCreateTemplate(ctx context.Context, rootDir, templateName, workspaceID string, noPrompt bool) error {
	inputs := createTemplateInputs{
		templateName: templateName,
		workspaceID:  workspaceID,
	}
	return runCreateTemplateWithInputs(ctx, rootDir, inputs, noPrompt)
}

func runCreateTemplateWithInputs(ctx context.Context, rootDir string, inputs createTemplateInputs, noPrompt bool) error {
	templateName := strings.TrimSpace(inputs.templateName)
	workspaceID := strings.TrimSpace(inputs.workspaceID)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	description := strings.TrimSpace(inputs.description)

	if templateName == "" || workspaceID == "" {
		if noPrompt {
			return fmt.Errorf("template name and workspace id are required without prompt")
		}
		var err error
		templateName, workspaceID, err = promptTemplateAndID(rootDir, "gws create", templateName, workspaceID, theme, useColor)
		if err != nil {
			return err
		}
	}
	if !noPrompt && !inputs.fromFlow {
		value, err := ui.PromptInputInline("description", "", nil, theme, useColor)
		if err != nil {
			return err
		}
		description = strings.TrimSpace(value)
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

	branches := inputs.branches
	if !noPrompt && !inputs.fromFlow {
		branches, err = promptTemplateBranches(ctx, tmpl, workspaceID, theme, useColor)
		if err != nil {
			return err
		}
	}

	startSteps(renderer)
	if err := ensureRepoGet(ctx, rootDir, missing, noPrompt, theme, useColor); err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}
	if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
		}
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
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestion(renderer, useColor, wsDir)
	return nil
}

func runCreateRepoWithInputs(ctx context.Context, rootDir string, inputs createRepoInputs, noPrompt bool) error {
	repoSpecs := template.NormalizeRepos(inputs.repos)
	workspaceID := strings.TrimSpace(inputs.workspaceID)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	description := strings.TrimSpace(inputs.description)

	if len(repoSpecs) == 0 {
		if noPrompt {
			return fmt.Errorf("repos are required without prompt")
		}
		choices, err := buildTemplateRepoChoices(rootDir)
		if err != nil {
			return err
		}
		if len(choices) == 0 {
			return fmt.Errorf("no repos found; run gws repo get first")
		}
		selected, err := ui.PromptChoiceSelect("gws create", "repo", choices, theme, useColor)
		if err != nil {
			return err
		}
		repoSpecs = template.NormalizeRepos([]string{selected})
	}
	if len(repoSpecs) != 1 {
		return fmt.Errorf("exactly one repo is required")
	}

	if workspaceID == "" {
		if noPrompt {
			return fmt.Errorf("workspace id is required without prompt")
		}
		value, err := ui.PromptInputInline("workspace id", "", func(v string) error {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("workspace id is required")
			}
			return nil
		}, theme, useColor)
		if err != nil {
			return err
		}
		workspaceID = strings.TrimSpace(value)
	}

	if !noPrompt && !inputs.fromFlow {
		value, err := ui.PromptInputInline("description", "", nil, theme, useColor)
		if err != nil {
			return err
		}
		description = strings.TrimSpace(value)
	}

	tmpl := template.Template{Repos: repoSpecs}
	missing, err := preflightTemplateRepos(ctx, rootDir, tmpl)
	if err != nil {
		return err
	}
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	branches := inputs.branches
	if !noPrompt && !inputs.fromFlow {
		branches, err = promptTemplateBranches(ctx, tmpl, workspaceID, theme, useColor)
		if err != nil {
			return err
		}
	}

	startSteps(renderer)
	if err := ensureRepoGet(ctx, rootDir, missing, noPrompt, theme, useColor); err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}
	if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	if err := applyTemplate(ctx, rootDir, workspaceID, tmpl, branches); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("apply repo selection failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
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
	wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderer.Blank()
	renderer.Section("Result")
	description := loadWorkspaceDescription(wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestion(renderer, useColor, workspace.WorkspaceDir(rootDir, workspaceID))
	return nil
}

func runCreateIssue(ctx context.Context, rootDir, issueURL, workspaceID, branch, baseRef string, noPrompt bool) error {
	issueURL = strings.TrimSpace(issueURL)
	workspaceID = strings.TrimSpace(workspaceID)
	branch = strings.TrimSpace(branch)
	baseRef = strings.TrimSpace(baseRef)

	if issueURL == "" {
		if noPrompt {
			return fmt.Errorf("issue URL is required when --no-prompt is set")
		}
		if workspaceID != "" || branch != "" || baseRef != "" {
			return fmt.Errorf("--workspace-id, --branch, and --base are only valid when an issue URL is provided")
		}
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("interactive issue picker requires a TTY")
		}
		return runIssuePicker(ctx, rootDir, noPrompt, "gws create")
	}

	req, err := parseIssueURL(issueURL)
	if err != nil {
		return err
	}
	repoURL := buildRepoURLFromParts(req.Host, req.Owner, req.Repo)
	description := ""
	if strings.EqualFold(strings.TrimSpace(req.Provider), "github") {
		issue, err := fetchGitHubIssue(ctx, req.Host, req.Owner, req.Repo, req.Number)
		if err != nil {
			return err
		}
		description = issue.Title
	}

	branchProvided := branch != ""
	if workspaceID == "" {
		workspaceID = formatIssueWorkspaceID(req.Owner, req.Repo, req.Number)
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
	store, err := repo.Open(ctx, rootDir, repoURL, true)
	if err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}
	if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
	if _, err := addIssueWorktree(ctx, rootDir, workspaceID, repoURL, branch, baseRef, store.StorePath); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("issue setup failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestion(renderer, useColor, wsDir)
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
		printCreateHelp(os.Stdout)
	}
	if err := issueFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printCreateHelp(os.Stdout)
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
		return runIssuePicker(ctx, rootDir, noPrompt, "gws create")
	}

	if issueFlags.NArg() != 1 {
		return fmt.Errorf("usage: gws create --issue [<ISSUE_URL>] [--workspace-id <id>] [--branch <name>] [--base <ref>]")
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
	description := ""
	if strings.EqualFold(strings.TrimSpace(req.Provider), "github") {
		issue, err := fetchGitHubIssue(ctx, req.Host, req.Owner, req.Repo, req.Number)
		if err != nil {
			return err
		}
		description = issue.Title
	}

	branchProvided := branch != ""
	if workspaceID == "" {
		workspaceID = formatIssueWorkspaceID(req.Owner, req.Repo, req.Number)
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
	store, err := repo.Open(ctx, rootDir, repoURL, true)
	if err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}
	if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
	if _, err := addIssueWorktree(ctx, rootDir, workspaceID, repoURL, branch, baseRef, store.StorePath); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("issue setup failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
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

func runIssuePicker(ctx context.Context, rootDir string, noPrompt bool, title string) error {
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

	repoSpec, err := ui.PromptChoiceSelect(title, "repo", promptChoices, theme, useColor)
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

	issueByNumber := make(map[int]issueSummary, len(issues))
	var issueChoices []ui.PromptChoice
	for _, issue := range issues {
		issueByNumber[issue.Number] = issue
		label := fmt.Sprintf("#%d", issue.Number)
		if strings.TrimSpace(issue.Title) != "" {
			label = fmt.Sprintf("#%d %s", issue.Number, strings.TrimSpace(issue.Title))
		}
		issueChoices = append(issueChoices, ui.PromptChoice{
			Label: label,
			Value: strconv.Itoa(issue.Number),
		})
	}

	validateBranch := func(value string) error {
		return workspace.ValidateBranchName(ctx, value)
	}
	selectedIssues, err := ui.PromptIssueSelectWithBranches(title, "issue", issueChoices, validateBranch, theme, useColor)
	if err != nil {
		return err
	}

	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

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
	store, err := repo.Open(ctx, rootDir, repoSpec, true)
	if err != nil {
		return err
	}

	repoURL := buildRepoURLFromParts(selectedRepo.Host, selectedRepo.Owner, selectedRepo.Repo)
	type issueWorkspaceResult struct {
		workspaceID string
		description string
		repos       []workspace.Repo
	}
	var results []issueWorkspaceResult
	var failure error
	var failureID string

	for _, selection := range selectedIssues {
		num, err := strconv.Atoi(strings.TrimSpace(selection.Value))
		if err != nil {
			failure = fmt.Errorf("invalid issue number: %s", selection.Value)
			failureID = selection.Value
			break
		}
		description := ""
		if issue, ok := issueByNumber[num]; ok {
			description = issue.Title
		}
		workspaceID := formatIssueWorkspaceID(selectedRepo.Owner, selectedRepo.Repo, num)
		branch := strings.TrimSpace(selection.Branch)
		if branch == "" {
			branch = fmt.Sprintf("issue/%d", num)
		}
		output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
		wsDir, err := workspace.New(ctx, rootDir, workspaceID)
		if err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
			} else {
				failure = err
			}
			failureID = workspaceID
			break
		}

		output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
		if _, err := addIssueWorktree(ctx, rootDir, workspaceID, repoURL, branch, "", store.StorePath); err != nil {
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
			description: description,
			repos:       repos,
		})
	}

	if len(results) > 0 {
		renderer.Blank()
		renderer.Section("Result")
		for _, result := range results {
			renderWorkspaceBlock(renderer, result.workspaceID, result.description, result.repos)
		}
	}
	if failure != nil {
		return fmt.Errorf("%s: %w", failureID, failure)
	}
	return nil
}

func runCreateIssueSelected(ctx context.Context, rootDir string, noPrompt bool, repoSpec string, selectedIssues []ui.IssueSelection) error {
	if strings.TrimSpace(repoSpec) == "" {
		return fmt.Errorf("repo is required")
	}
	if len(selectedIssues) == 0 {
		return fmt.Errorf("at least one issue is required")
	}
	repoChoices, err := buildIssueRepoChoices(rootDir)
	if err != nil {
		return err
	}
	_, repoByValue := toIssuePromptChoices(repoChoices)
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

	issueByNumber := make(map[int]issueSummary, len(issues))
	for _, issue := range issues {
		issueByNumber[issue.Number] = issue
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

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
	store, err := repo.Open(ctx, rootDir, repoSpec, true)
	if err != nil {
		return err
	}

	repoURL := buildRepoURLFromParts(selectedRepo.Host, selectedRepo.Owner, selectedRepo.Repo)
	type issueWorkspaceResult struct {
		workspaceID string
		description string
		repos       []workspace.Repo
	}
	var results []issueWorkspaceResult
	var failure error
	var failureID string

	for _, selection := range selectedIssues {
		num, err := strconv.Atoi(strings.TrimSpace(selection.Value))
		if err != nil {
			failure = fmt.Errorf("invalid issue number: %s", selection.Value)
			failureID = selection.Value
			break
		}
		description := ""
		if issue, ok := issueByNumber[num]; ok {
			description = issue.Title
		}
		workspaceID := formatIssueWorkspaceID(selectedRepo.Owner, selectedRepo.Repo, num)
		branch := strings.TrimSpace(selection.Branch)
		if branch == "" {
			branch = fmt.Sprintf("issue/%d", num)
		}
		if err := workspace.ValidateBranchName(ctx, branch); err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
		wsDir, err := workspace.New(ctx, rootDir, workspaceID)
		if err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
			} else {
				failure = err
			}
			failureID = workspaceID
			break
		}

		output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
		if _, err := addIssueWorktree(ctx, rootDir, workspaceID, repoURL, branch, "", store.StorePath); err != nil {
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
			description: description,
			repos:       repos,
		})
	}

	if len(results) > 0 {
		renderer.Blank()
		renderer.Section("Result")
		for _, result := range results {
			renderWorkspaceBlock(renderer, result.workspaceID, result.description, result.repos)
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

func toIssuePromptChoices(choices []issueRepoChoice) ([]ui.PromptChoice, map[string]issueRepoChoice) {
	prompt := make([]ui.PromptChoice, 0, len(choices))
	byValue := make(map[string]issueRepoChoice, len(choices))
	for _, choice := range choices {
		prompt = append(prompt, ui.PromptChoice{Label: choice.Label, Value: choice.Value})
		byValue[choice.Value] = choice
	}
	return prompt, byValue
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

func fetchGitHubIssue(ctx context.Context, host, owner, repoName string, number int) (issueSummary, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" || number <= 0 {
		return issueSummary{}, fmt.Errorf("owner/repo and issue number are required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/issues/%d", owner, repoName, number)
	args := []string{"api", "-X", "GET", endpoint}
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
			return issueSummary{}, fmt.Errorf("gh api failed: %s", msg)
		}
		return issueSummary{}, fmt.Errorf("gh api failed: %w", err)
	}
	var item githubIssueItem
	if err := json.Unmarshal(stdout.Bytes(), &item); err != nil {
		return issueSummary{}, fmt.Errorf("parse gh api response: %w", err)
	}
	if item.Number == 0 {
		return issueSummary{}, fmt.Errorf("issue not found")
	}
	return issueSummary{
		Number: item.Number,
		Title:  strings.TrimSpace(item.Title),
	}, nil
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

func buildIssueChoices(issues []issueSummary) []ui.PromptChoice {
	var choices []ui.PromptChoice
	for _, issue := range issues {
		label := fmt.Sprintf("#%d", issue.Number)
		if strings.TrimSpace(issue.Title) != "" {
			label = fmt.Sprintf("#%d %s", issue.Number, strings.TrimSpace(issue.Title))
		}
		choices = append(choices, ui.PromptChoice{
			Label: label,
			Value: strconv.Itoa(issue.Number),
		})
	}
	return choices
}

func buildPRChoices(prs []prSummary) []ui.PromptChoice {
	var choices []ui.PromptChoice
	for _, pr := range prs {
		label := fmt.Sprintf("#%d", pr.Number)
		if strings.TrimSpace(pr.Title) != "" {
			label = fmt.Sprintf("#%d %s", pr.Number, strings.TrimSpace(pr.Title))
		}
		choices = append(choices, ui.PromptChoice{
			Label: label,
			Value: strconv.Itoa(pr.Number),
		})
	}
	return choices
}

type reviewRepoChoice struct {
	Label   string
	Value   string
	Host    string
	Owner   string
	Repo    string
	RepoURL string
}

type prSummary struct {
	Number   int
	Title    string
	HeadRef  string
	HeadRepo string
	BaseRepo string
}

func runCreateReview(ctx context.Context, rootDir, prURL string, noPrompt bool) error {
	prURL = strings.TrimSpace(prURL)
	if prURL == "" {
		if noPrompt {
			return fmt.Errorf("PR URL is required when --no-prompt is set")
		}
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("interactive review picker requires a TTY")
		}
		return runCreateReviewPicker(ctx, rootDir, noPrompt)
	}

	req, err := parsePRURL(prURL)
	if err != nil {
		return err
	}

	pr, err := fetchGitHubPR(ctx, req.Host, req.Owner, req.Repo, req.Number)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(pr.HeadRepo), strings.TrimSpace(pr.BaseRepo)) {
		return fmt.Errorf("fork PRs are not supported: %s", pr.HeadRepo)
	}
	description := pr.Title

	baseOwner, baseRepo, ok := splitRepoFullName(pr.BaseRepo)
	if !ok {
		return fmt.Errorf("invalid base repo: %s", pr.BaseRepo)
	}
	repoURL := buildRepoURLFromParts(req.Host, baseOwner, baseRepo)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Inputs")
	renderer.Bullet(fmt.Sprintf("provider: github (%s)", req.Host))
	renderer.Bullet(fmt.Sprintf("repo: %s/%s", baseOwner, baseRepo))
	renderer.Bullet(fmt.Sprintf("pull request: #%d", pr.Number))
	renderer.Bullet(fmt.Sprintf("branch: %s", pr.HeadRef))
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

	workspaceID := formatReviewWorkspaceID(baseOwner, baseRepo, pr.Number)
	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}
	if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	store, err := repo.Open(ctx, rootDir, repoURL, false)
	if err != nil {
		return err
	}
	if err := fetchPRHead(ctx, store.StorePath, pr.HeadRef); err != nil {
		return err
	}

	headRef := fmt.Sprintf("refs/remotes/origin/%s", pr.HeadRef)
	output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
	if _, err := workspace.AddWithTrackingBranch(ctx, rootDir, workspaceID, repoURL, "", pr.HeadRef, headRef, false); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("review failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestion(renderer, useColor, wsDir)
	return nil
}

func runCreateReviewPicker(ctx context.Context, rootDir string, noPrompt bool) error {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	repoChoices, err := buildReviewRepoChoices(rootDir)
	if err != nil {
		return err
	}
	if len(repoChoices) == 0 {
		return fmt.Errorf("no GitHub repos found")
	}

	promptChoices := make([]ui.PromptChoice, 0, len(repoChoices))
	repoByValue := make(map[string]reviewRepoChoice, len(repoChoices))
	for _, choice := range repoChoices {
		promptChoices = append(promptChoices, ui.PromptChoice{Label: choice.Label, Value: choice.Value})
		repoByValue[choice.Value] = choice
	}

	repoSpec, err := ui.PromptChoiceSelect("gws create", "repo", promptChoices, theme, useColor)
	if err != nil {
		return err
	}
	selectedRepo, ok := repoByValue[repoSpec]
	if !ok {
		return fmt.Errorf("selected repo not found")
	}

	prs, err := fetchGitHubPRs(ctx, selectedRepo.Host, selectedRepo.Owner, selectedRepo.Repo)
	if err != nil {
		return err
	}
	if len(prs) == 0 {
		return fmt.Errorf("no pull requests found")
	}

	var prChoices []ui.PromptChoice
	for _, pr := range prs {
		label := fmt.Sprintf("#%d", pr.Number)
		if strings.TrimSpace(pr.Title) != "" {
			label = fmt.Sprintf("#%d %s", pr.Number, strings.TrimSpace(pr.Title))
		}
		prChoices = append(prChoices, ui.PromptChoice{
			Label: label,
			Value: strconv.Itoa(pr.Number),
		})
	}

	selectedPRs, err := ui.PromptMultiSelect("gws create", "pull request", prChoices, theme, useColor)
	if err != nil {
		return err
	}

	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Steps")

	_, exists, err := repo.Exists(rootDir, selectedRepo.RepoURL)
	if err != nil {
		return err
	}
	if !exists {
		if err := ensureRepoGet(ctx, rootDir, []string{selectedRepo.RepoURL}, noPrompt, theme, useColor); err != nil {
			return err
		}
	}

	prByNumber := make(map[int]prSummary, len(prs))
	for _, pr := range prs {
		prByNumber[pr.Number] = pr
	}

	type reviewWorkspaceResult struct {
		workspaceID string
		description string
		repos       []workspace.Repo
	}
	var results []reviewWorkspaceResult
	var failure error
	var failureID string

	for _, value := range selectedPRs {
		num, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			failure = fmt.Errorf("invalid PR number: %s", value)
			failureID = value
			break
		}
		pr, ok := prByNumber[num]
		if !ok {
			failure = fmt.Errorf("PR not found: %d", num)
			failureID = fmt.Sprintf("PR-%d", num)
			break
		}
		if !strings.EqualFold(strings.TrimSpace(pr.HeadRepo), strings.TrimSpace(pr.BaseRepo)) {
			failure = fmt.Errorf("fork PRs are not supported: %s", pr.HeadRepo)
			failureID = fmt.Sprintf("PR-%d", num)
			break
		}
		description := pr.Title
		workspaceID := formatReviewWorkspaceID(selectedRepo.Owner, selectedRepo.Repo, num)
		output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
		wsDir, err := workspace.New(ctx, rootDir, workspaceID)
		if err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
			} else {
				failure = err
			}
			failureID = workspaceID
			break
		}

		store, err := repo.Open(ctx, rootDir, selectedRepo.RepoURL, false)
		if err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		if err := fetchPRHead(ctx, store.StorePath, pr.HeadRef); err != nil {
			failure = err
			failureID = workspaceID
			break
		}

		headRef := fmt.Sprintf("refs/remotes/origin/%s", pr.HeadRef)
		output.Step(formatStep("worktree add", displayRepoName(selectedRepo.RepoURL), worktreeDest(rootDir, workspaceID, selectedRepo.RepoURL)))
		if _, err := workspace.AddWithTrackingBranch(ctx, rootDir, workspaceID, selectedRepo.RepoURL, "", pr.HeadRef, headRef, false); err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = fmt.Errorf("review failed: %w (rollback failed: %v)", err, rollbackErr)
			} else {
				failure = err
			}
			failureID = workspaceID
			break
		}

		repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
		results = append(results, reviewWorkspaceResult{
			workspaceID: workspaceID,
			description: description,
			repos:       repos,
		})
	}

	if len(results) > 0 {
		renderer.Blank()
		renderer.Section("Result")
		for _, result := range results {
			renderWorkspaceBlock(renderer, result.workspaceID, result.description, result.repos)
		}
	}
	if failure != nil {
		return fmt.Errorf("%s: %w", failureID, failure)
	}
	return nil
}

func runCreateReviewSelected(ctx context.Context, rootDir string, noPrompt bool, repoSpec string, selectedPRs []string) error {
	if strings.TrimSpace(repoSpec) == "" {
		return fmt.Errorf("repo is required")
	}
	if len(selectedPRs) == 0 {
		return fmt.Errorf("at least one pull request is required")
	}
	repoChoices, err := buildReviewRepoChoices(rootDir)
	if err != nil {
		return err
	}
	_, repoByValue := toPromptChoices(repoChoices)
	selectedRepo, ok := repoByValue[repoSpec]
	if !ok {
		return fmt.Errorf("selected repo not found")
	}
	prs, err := fetchGitHubPRs(ctx, selectedRepo.Host, selectedRepo.Owner, selectedRepo.Repo)
	if err != nil {
		return err
	}
	if len(prs) == 0 {
		return fmt.Errorf("no pull requests found")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Steps")

	_, exists, err := repo.Exists(rootDir, selectedRepo.RepoURL)
	if err != nil {
		return err
	}
	if !exists {
		if err := ensureRepoGet(ctx, rootDir, []string{selectedRepo.RepoURL}, noPrompt, theme, useColor); err != nil {
			return err
		}
	}

	prByNumber := make(map[int]prSummary, len(prs))
	for _, pr := range prs {
		prByNumber[pr.Number] = pr
	}

	type reviewWorkspaceResult struct {
		workspaceID string
		description string
		repos       []workspace.Repo
	}
	var results []reviewWorkspaceResult
	var failure error
	var failureID string

	for _, value := range selectedPRs {
		num, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			failure = fmt.Errorf("invalid PR number: %s", value)
			failureID = value
			break
		}
		pr, ok := prByNumber[num]
		if !ok {
			failure = fmt.Errorf("PR not found: %d", num)
			failureID = fmt.Sprintf("PR-%d", num)
			break
		}
		if !strings.EqualFold(strings.TrimSpace(pr.HeadRepo), strings.TrimSpace(pr.BaseRepo)) {
			failure = fmt.Errorf("fork PRs are not supported: %s", pr.HeadRepo)
			failureID = fmt.Sprintf("PR-%d", num)
			break
		}
		description := pr.Title
		workspaceID := formatReviewWorkspaceID(selectedRepo.Owner, selectedRepo.Repo, num)
		output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
		wsDir, err := workspace.New(ctx, rootDir, workspaceID)
		if err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
			} else {
				failure = err
			}
			failureID = workspaceID
			break
		}

		store, err := repo.Open(ctx, rootDir, selectedRepo.RepoURL, false)
		if err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		if err := fetchPRHead(ctx, store.StorePath, pr.HeadRef); err != nil {
			failure = err
			failureID = workspaceID
			break
		}

		headRef := fmt.Sprintf("refs/remotes/origin/%s", pr.HeadRef)
		output.Step(formatStep("worktree add", displayRepoName(selectedRepo.RepoURL), worktreeDest(rootDir, workspaceID, selectedRepo.RepoURL)))
		if _, err := workspace.AddWithTrackingBranch(ctx, rootDir, workspaceID, selectedRepo.RepoURL, "", pr.HeadRef, headRef, false); err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = fmt.Errorf("review failed: %w (rollback failed: %v)", err, rollbackErr)
			} else {
				failure = err
			}
			failureID = workspaceID
			break
		}

		repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
		results = append(results, reviewWorkspaceResult{
			workspaceID: workspaceID,
			description: description,
			repos:       repos,
		})
	}

	if len(results) > 0 {
		renderer.Blank()
		renderer.Section("Result")
		for _, result := range results {
			renderWorkspaceBlock(renderer, result.workspaceID, result.description, result.repos)
		}
	}
	if failure != nil {
		return fmt.Errorf("%s: %w", failureID, failure)
	}
	return nil
}

func buildReviewRepoChoices(rootDir string) ([]reviewRepoChoice, error) {
	repos, _, err := repo.List(rootDir)
	if err != nil {
		return nil, err
	}
	var choices []reviewRepoChoice
	for _, entry := range repos {
		repoKey := displayRepoKey(entry.RepoKey)
		parts := strings.Split(repoKey, "/")
		if len(parts) < 3 {
			continue
		}
		host := parts[0]
		owner := parts[1]
		repoName := parts[2]
		if !isGitHubHost(host) {
			continue
		}
		label := fmt.Sprintf("%s (%s/%s)", repoName, owner, repoName)
		repoURL := buildRepoURLFromParts(host, owner, repoName)
		value := repoSpecFromKey(entry.RepoKey)
		choices = append(choices, reviewRepoChoice{
			Label:   label,
			Value:   value,
			Host:    host,
			Owner:   owner,
			Repo:    repoName,
			RepoURL: repoURL,
		})
	}
	return choices, nil
}

func toPromptChoices(choices []reviewRepoChoice) ([]ui.PromptChoice, map[string]reviewRepoChoice) {
	prompt := make([]ui.PromptChoice, 0, len(choices))
	byValue := make(map[string]reviewRepoChoice, len(choices))
	for _, choice := range choices {
		prompt = append(prompt, ui.PromptChoice{Label: choice.Label, Value: choice.Value})
		byValue[choice.Value] = choice
	}
	return prompt, byValue
}

func isGitHubHost(host string) bool {
	lower := strings.ToLower(strings.TrimSpace(host))
	return strings.Contains(lower, "github")
}

type githubPRItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Head   struct {
		Ref  string `json:"ref"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
}

func fetchGitHubPR(ctx context.Context, host, owner, repoName string, number int) (prSummary, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" || number <= 0 {
		return prSummary{}, fmt.Errorf("owner/repo and PR number are required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repoName, number)
	args := []string{"api", "-X", "GET", endpoint}
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
			return prSummary{}, fmt.Errorf("gh api failed: %s", msg)
		}
		return prSummary{}, fmt.Errorf("gh api failed: %w", err)
	}
	var item githubPRItem
	if err := json.Unmarshal(stdout.Bytes(), &item); err != nil {
		return prSummary{}, fmt.Errorf("parse gh api response: %w", err)
	}
	return normalizeGitHubPR(item), nil
}

func fetchGitHubPRs(ctx context.Context, host, owner, repoName string) ([]prSummary, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" {
		return nil, fmt.Errorf("owner/repo is required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/pulls", owner, repoName)
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
	return parseGitHubPRs(stdout.Bytes())
}

func parseGitHubPRs(data []byte) ([]prSummary, error) {
	var raw []githubPRItem
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gh api response: %w", err)
	}
	var prs []prSummary
	for _, item := range raw {
		if item.Number == 0 {
			continue
		}
		prs = append(prs, normalizeGitHubPR(item))
	}
	return prs, nil
}

func normalizeGitHubPR(item githubPRItem) prSummary {
	return prSummary{
		Number:   item.Number,
		Title:    strings.TrimSpace(item.Title),
		HeadRef:  strings.TrimSpace(item.Head.Ref),
		HeadRepo: strings.TrimSpace(item.Head.Repo.FullName),
		BaseRepo: strings.TrimSpace(item.Base.Repo.FullName),
	}
}

func splitRepoFullName(fullName string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(fullName), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func formatReviewWorkspaceID(owner, repo string, number int) string {
	return fmt.Sprintf("%s-%s-REVIEW-PR-%d", strings.ToUpper(strings.TrimSpace(owner)), strings.ToUpper(strings.TrimSpace(repo)), number)
}

func formatIssueWorkspaceID(owner, repo string, number int) string {
	return fmt.Sprintf("%s-%s-ISSUE-%d", strings.ToUpper(strings.TrimSpace(owner)), strings.ToUpper(strings.TrimSpace(repo)), number)
}

func fetchPRHead(ctx context.Context, storePath, headRef string) error {
	if strings.TrimSpace(storePath) == "" {
		return fmt.Errorf("store path is required")
	}
	if strings.TrimSpace(headRef) == "" {
		return fmt.Errorf("head ref is required")
	}
	gitcmd.Logf("git fetch origin %s", headRef)
	if _, err := gitcmd.Run(ctx, []string{"fetch", "origin", headRef}, gitcmd.Options{Dir: storePath}); err != nil {
		return err
	}
	return nil
}

func localBranchExists(ctx context.Context, storePath, branch string) (bool, error) {
	if strings.TrimSpace(storePath) == "" {
		return false, fmt.Errorf("store path is required")
	}
	if strings.TrimSpace(branch) == "" {
		return false, fmt.Errorf("branch is required")
	}
	ref := fmt.Sprintf("refs/heads/%s", branch)
	_, exists, err := gitcmd.ShowRef(ctx, storePath, ref)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func remoteBranchExists(ctx context.Context, storePath, branch string) (bool, error) {
	if strings.TrimSpace(storePath) == "" {
		return false, fmt.Errorf("store path is required")
	}
	if strings.TrimSpace(branch) == "" {
		return false, fmt.Errorf("branch is required")
	}
	remoteRef := fmt.Sprintf("refs/remotes/origin/%s", branch)
	if _, exists, err := gitcmd.ShowRef(ctx, storePath, remoteRef); err != nil {
		return false, err
	} else if exists {
		return true, nil
	}
	res, err := gitcmd.Run(ctx, []string{"ls-remote", "--heads", "origin", branch}, gitcmd.Options{Dir: storePath})
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(res.Stdout) != "", nil
}

func fetchRemoteBranch(ctx context.Context, storePath, branch string) error {
	if strings.TrimSpace(storePath) == "" {
		return fmt.Errorf("store path is required")
	}
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("branch is required")
	}
	gitcmd.Logf("git fetch origin %s", branch)
	if _, err := gitcmd.Run(ctx, []string{"fetch", "origin", branch}, gitcmd.Options{Dir: storePath}); err != nil {
		return err
	}
	return nil
}

func addIssueWorktree(ctx context.Context, rootDir, workspaceID, repoURL, branch, baseRef, storePath string) (workspace.Repo, error) {
	if strings.TrimSpace(storePath) == "" {
		return workspace.Repo{}, fmt.Errorf("store path is required")
	}
	localExists, err := localBranchExists(ctx, storePath, branch)
	if err != nil {
		return workspace.Repo{}, err
	}
	if !localExists {
		remoteExists, err := remoteBranchExists(ctx, storePath, branch)
		if err != nil {
			return workspace.Repo{}, err
		}
		if remoteExists {
			if err := fetchRemoteBranch(ctx, storePath, branch); err != nil {
				return workspace.Repo{}, err
			}
			remoteRef := fmt.Sprintf("refs/remotes/origin/%s", branch)
			return workspace.AddWithTrackingBranch(ctx, rootDir, workspaceID, repoURL, "", branch, remoteRef, false)
		}
	}
	return workspace.AddWithBranch(ctx, rootDir, workspaceID, repoURL, "", branch, baseRef, true)
}

func formatPRList(values []string) string {
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
		printCreateHelp(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gws create --review [<PR URL>]")
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
	pr, err := fetchGitHubPR(ctx, req.Host, req.Owner, req.Repo, req.Number)
	if err != nil {
		return err
	}
	description := pr.Title

	_, exists, err := repo.Exists(rootDir, repoURL)
	if err != nil {
		return err
	}
	workspaceID := formatReviewWorkspaceID(req.Owner, req.Repo, req.Number)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Inputs")
	renderer.Bullet(fmt.Sprintf("provider: %s (%s)", strings.ToLower(req.Provider), req.Host))
	renderer.Blank()
	renderer.Section("Steps")

	if !exists {
		if err := ensureRepoGet(ctx, rootDir, []string{repoURL}, noPrompt, theme, useColor); err != nil {
			return err
		}
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}
	if err := workspace.SaveMetadata(wsDir, workspace.Metadata{Description: description}); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
		}
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
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
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
	if host == "" {
		return prRequest{}, fmt.Errorf("invalid PR URL host: %s", raw)
	}
	if !isGitHubHost(host) {
		return prRequest{}, fmt.Errorf("unsupported PR host: %s", host)
	}
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

func promptTemplateAndID(rootDir, title, templateName, workspaceID string, theme ui.Theme, useColor bool) (string, string, error) {
	file, err := template.Load(rootDir)
	if err != nil {
		return "", "", err
	}
	names := template.Names(file)
	if len(names) == 0 {
		return "", "", fmt.Errorf("no templates found in %s", filepath.Join(rootDir, template.FileName))
	}
	templateName, workspaceID, err = ui.PromptNewWorkspaceInputs(title, names, templateName, workspaceID, theme, useColor)
	if err != nil {
		return "", "", err
	}
	return templateName, workspaceID, nil
}

func promptCreateMode(theme ui.Theme, useColor bool) (string, error) {
	choices := []ui.PromptChoice{
		{Label: "repo", Value: "repo", Description: "1 repo only"},
		{Label: "issue", Value: "issue", Description: "From an issue (multi-select)"},
		{Label: "review", Value: "review", Description: "From a review request (multi-select)"},
		{Label: "template", Value: "template", Description: "From template"},
	}
	return ui.PromptChoiceSelect("gws create", "mode", choices, theme, useColor)
}

func loadTemplateNames(rootDir string) ([]string, error) {
	file, err := template.Load(rootDir)
	if err != nil {
		return nil, err
	}
	names := template.Names(file)
	if len(names) == 0 {
		return nil, fmt.Errorf("no templates found in %s", filepath.Join(rootDir, template.FileName))
	}
	return names, nil
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
	if r == nil {
		return
	}
	for i, repo := range repos {
		prefix := "├─ "
		if i == len(repos)-1 {
			prefix = "└─ "
		}
		name := formatRepoName(repo.Alias, repo.RepoKey)
		r.TreeLineBranchMuted(extraIndent+prefix, name, repo.Branch)
	}
}

func renderWorkspaceBlock(r *ui.Renderer, workspaceID, description string, repos []workspace.Repo) {
	if r == nil {
		return
	}
	r.BulletWithDescription(workspaceID, description, fmt.Sprintf("(repos: %d)", len(repos)))
	renderWorkspaceRepos(r, repos, output.Indent)
}

func loadWorkspaceDescription(wsDir string) string {
	desc, err := workspace.ReadDescription(wsDir)
	if err != nil {
		return ""
	}
	return desc
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
	repos, _, err := workspace.ScanRepos(ctx, entry.WorkspacePath)
	if err != nil {
		return buildWorkspaceChoiceFromRepos(entry, nil)
	}
	return buildWorkspaceChoiceFromRepos(entry, repos)
}

func buildWorkspaceChoiceFromRepos(entry workspace.Entry, repos []workspace.Repo) ui.WorkspaceChoice {
	choice := ui.WorkspaceChoice{
		ID:          entry.WorkspaceID,
		Description: entry.Description,
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
	return repo.DisplaySpec(repoSpec)
}

func displayRepoSpec(repoSpec string) string {
	return displayTemplateRepo(repoSpec)
}

func displayRepoName(repoSpec string) string {
	return repo.DisplayName(repoSpec)
}

func repoDestForSpec(rootDir, repoSpec string) string {
	store := repoStoreRel(rootDir, repoSpec)
	return store
}

func repoStoreRel(rootDir, repoSpec string) string {
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		return ""
	}
	storePath := repo.StorePath(rootDir, spec)
	return relPath(rootDir, storePath)
}

func worktreeDest(rootDir, workspaceID, repoSpec string) string {
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil || spec.Repo == "" {
		return ""
	}
	wsPath := workspace.WorktreePath(rootDir, workspaceID, spec.Repo)
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

func removeConfirmLabel(state workspace.WorkspaceState) string {
	switch state.Kind {
	case workspace.WorkspaceStateDirty:
		return "This workspace has uncommitted changes. Remove anyway?"
	case workspace.WorkspaceStateUnpushed:
		return "This workspace has unpushed commits. Remove anyway?"
	case workspace.WorkspaceStateDiverged:
		return "This workspace has diverged from upstream. Remove anyway?"
	case workspace.WorkspaceStateUnknown:
		return "Workspace status could not be read. Remove anyway?"
	default:
		return "Remove workspace?"
	}
}

func workspaceRemoveWarningLabel(state workspace.WorkspaceState) (string, bool) {
	switch state.Kind {
	case workspace.WorkspaceStateUnpushed:
		return "unpushed commits", false
	case workspace.WorkspaceStateDiverged:
		return "diverged or upstream missing", false
	case workspace.WorkspaceStateUnknown:
		return "status unknown", true
	case workspace.WorkspaceStateDirty:
		return "dirty changes", true
	default:
		return "", false
	}
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

func loadWorkspaceStateForRemoval(ctx context.Context, rootDir, workspaceID string) workspace.WorkspaceState {
	state, err := workspace.State(ctx, rootDir, workspaceID)
	if err == nil {
		return state
	}
	return workspace.WorkspaceState{
		WorkspaceID: workspaceID,
		Kind:        workspace.WorkspaceStateUnknown,
		Warnings:    []error{err},
	}
}

func classifyWorkspaceRemoval(ctx context.Context, rootDir string, entries []workspace.Entry) ([]ui.WorkspaceChoice, []ui.BlockedChoice) {
	var removable []ui.WorkspaceChoice
	for _, entry := range entries {
		state := loadWorkspaceStateForRemoval(ctx, rootDir, entry.WorkspaceID)
		choice := buildWorkspaceChoice(ctx, entry)
		choice.Warning, choice.WarningStrong = workspaceRemoveWarningLabel(state)
		removable = append(removable, choice)
	}
	return removable, nil
}

func buildWorkspaceRemoveReason(state workspace.WorkspaceState) string {
	var dirtyRepos []string
	var reasons []string
	for _, repo := range state.Repos {
		name := strings.TrimSpace(repo.Alias)
		if name == "" {
			name = "unknown"
		}
		if repo.Kind != workspace.RepoStateDirty {
			continue
		}
		detail := formatDirtySummaryCounts(repo.StagedCount, repo.UnstagedCount, repo.UntrackedCount, repo.UnmergedCount)
		if detail == "" {
			detail = "dirty"
		}
		dirtyRepos = append(dirtyRepos, fmt.Sprintf("%s (%s)", name, detail))
	}
	if len(dirtyRepos) > 0 {
		reasons = append(reasons, fmt.Sprintf("dirty: %s", strings.Join(dirtyRepos, ", ")))
	}
	return strings.Join(reasons, "; ")
}

func formatDirtySummary(repo workspace.RepoStatus) string {
	return formatDirtySummaryCounts(repo.StagedCount, repo.UnstagedCount, repo.UntrackedCount, repo.UnmergedCount)
}

func formatDirtySummaryCounts(staged, unstaged, untracked, unmerged int) string {
	var parts []string
	if staged > 0 {
		parts = append(parts, fmt.Sprintf("staged=%d", staged))
	}
	if unstaged > 0 {
		parts = append(parts, fmt.Sprintf("unstaged=%d", unstaged))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("untracked=%d", untracked))
	}
	if unmerged > 0 {
		parts = append(parts, fmt.Sprintf("unmerged=%d", unmerged))
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
	writeWorkspaceListText(ctx, rootDir, entries, warnings)
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

	var selected []string
	selectedFromPrompt := false
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
		removable, _ := classifyWorkspaceRemoval(ctx, rootDir, workspaces)
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		if len(removable) == 0 {
			renderer := ui.NewRenderer(os.Stdout, theme, useColor)
			renderer.Section("Info")
			renderer.Bullet("no removable workspaces")
			return fmt.Errorf("no removable workspaces")
		}
		selected, err = ui.PromptWorkspaceMultiSelectWithBlocked("gws rm", removable, nil, theme, useColor)
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			return nil
		}
		selectedFromPrompt = true
		if len(selected) == 1 {
			workspaceID = selected[0]
		}
	} else {
		selected = []string{workspaceID}
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	if len(selected) == 1 {
		state := loadWorkspaceStateForRemoval(ctx, rootDir, workspaceID)
		if !selectedFromPrompt && state.Kind != workspace.WorkspaceStateClean {
			label := removeConfirmLabel(state)
			confirm, err := ui.PromptConfirmInline(label, theme, useColor)
			if err != nil {
				return err
			}
			if !confirm {
				return nil
			}
		}
		renderer.Section("Steps")
		output.Step(formatStep("remove workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))

		if err := workspace.RemoveWithOptions(ctx, rootDir, workspaceID, workspace.RemoveOptions{
			AllowStatusError: true,
			AllowDirty:       state.Kind == workspace.WorkspaceStateDirty,
		}); err != nil {
			return err
		}

		renderer.Blank()
		renderer.Section("Result")
		renderer.Bullet(fmt.Sprintf("%s removed", workspaceID))
		return nil
	}

	requiresConfirm := false
	requiresStrongConfirm := false
	states := make(map[string]workspace.WorkspaceState, len(selected))
	for _, selectedID := range selected {
		state := loadWorkspaceStateForRemoval(ctx, rootDir, selectedID)
		states[selectedID] = state
		if state.Kind != workspace.WorkspaceStateClean {
			requiresConfirm = true
		}
		if state.Kind == workspace.WorkspaceStateDirty || state.Kind == workspace.WorkspaceStateUnknown {
			requiresStrongConfirm = true
		}
	}
	if !selectedFromPrompt {
		confirmLabel := fmt.Sprintf("Remove %d workspaces?", len(selected))
		if requiresStrongConfirm {
			confirmLabel = fmt.Sprintf("Selected workspaces include uncommitted changes or status errors. Remove %d workspaces anyway?", len(selected))
		} else if requiresConfirm {
			confirmLabel = fmt.Sprintf("Selected workspaces have warnings. Remove %d workspaces anyway?", len(selected))
		}
		confirm, err := ui.PromptConfirmInline(confirmLabel, theme, useColor)
		if err != nil {
			return err
		}
		if !confirm {
			return nil
		}
	}

	renderer.Section("Steps")
	for i, selectedID := range selected {
		output.Step(formatStepWithIndex("remove workspace", selectedID, relPath(rootDir, workspace.WorkspaceDir(rootDir, selectedID)), i+1, len(selected)))
		state := states[selectedID]
		if err := workspace.RemoveWithOptions(ctx, rootDir, selectedID, workspace.RemoveOptions{
			AllowStatusError: true,
			AllowDirty:       state.Kind == workspace.WorkspaceStateDirty,
		}); err != nil {
			return err
		}
	}

	renderer.Blank()
	renderer.Section("Result")
	for _, selectedID := range selected {
		renderer.Bullet(fmt.Sprintf("%s removed", selectedID))
	}
	return nil
}

func writeWorkspaceStatusText(result workspace.StatusResult) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	warningLines := appendWarningLines(nil, "", result.Warnings)
	for _, repo := range result.Repos {
		if repo.Error != nil {
			label := strings.TrimSpace(repo.Alias)
			if label == "" {
				label = filepath.Base(repo.WorktreePath)
			}
			warningLines = append(warningLines, fmt.Sprintf("%s: %v", label, repo.Error))
		}
	}
	if len(warningLines) > 0 {
		renderWarningsSection(renderer, "warnings", warningLines, false)
		renderer.Blank()
	}

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
	}
}

func writeWorkspaceListText(ctx context.Context, rootDir string, entries []workspace.Entry, warnings []error) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	type workspaceListEntry struct {
		entry workspace.Entry
		repos []workspace.Repo
		state workspace.WorkspaceState
	}
	var items []workspaceListEntry
	var repoWarnings []string
	for _, entry := range entries {
		repos, warnings, err := workspace.ScanRepos(ctx, entry.WorkspacePath)
		if err != nil {
			repoWarnings = append(repoWarnings, fmt.Sprintf("%s: %s", entry.WorkspaceID, compactError(err)))
		}
		repoWarnings = appendWarningLines(repoWarnings, entry.WorkspaceID, warnings)
		state := loadWorkspaceStateForRemoval(ctx, rootDir, entry.WorkspaceID)
		items = append(items, workspaceListEntry{entry: entry, repos: repos, state: state})
	}
	repoWarnings = appendWarningLines(repoWarnings, "", warnings)
	if len(repoWarnings) > 0 {
		renderWarningsSection(renderer, "warnings", repoWarnings, false)
		renderer.Blank()
	}

	renderer.Section("Result")
	var choices []ui.WorkspaceChoice
	for _, item := range items {
		choice := buildWorkspaceChoiceFromRepos(item.entry, item.repos)
		choice.Warning, choice.WarningStrong = workspaceRemoveWarningLabel(item.state)
		choices = append(choices, choice)
	}
	for _, line := range ui.WorkspaceChoiceLines(choices, -1, useColor, theme) {
		renderer.LineRaw(line)
	}
}

func writeRepoListText(entries []repo.Entry, warnings []error) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	warningLines := appendWarningLines(nil, "", warnings)
	if len(warningLines) > 0 {
		renderWarningsSection(renderer, "warnings", warningLines, false)
		renderer.Blank()
	}

	renderer.Section("Result")
	for _, entry := range entries {
		renderer.Bullet(fmt.Sprintf("%s %s", entry.RepoKey, entry.StorePath))
	}
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
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Result")
	renderer.Bullet(name)
	if len(tmpl.Repos) > 0 {
		renderTreeLines(renderer, tmpl.Repos, treeLineNormal)
	}
}

func writeInitText(result initcmd.Result) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	var skipped []string
	for _, dir := range result.SkippedDirs {
		skipped = append(skipped, fmt.Sprintf("dir: %s", dir))
	}
	for _, file := range result.SkippedFiles {
		skipped = append(skipped, fmt.Sprintf("file: %s", file))
	}
	if len(skipped) > 0 {
		renderer.Section("Info")
		renderer.Bullet("already exists")
		renderTreeLines(renderer, skipped, treeLineNormal)
		renderer.Blank()
	}

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
	renderer.Bullet(fmt.Sprintf("root: %s", result.RootDir))

	renderSuggestions(renderer, useColor, []string{
		fmt.Sprintf("Edit templates.yaml: %s", filepath.Join(result.RootDir, "templates.yaml")),
	})
}
func writeDoctorText(result doctor.Result, fixed []string) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	if len(result.Warnings) > 0 {
		var lines []string
		for _, warning := range result.Warnings {
			lines = append(lines, warning.Error())
		}
		renderWarningsSection(renderer, "warnings", lines, false)
		renderer.Blank()
	}

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

}

func writeDoctorSelfText(result doctor.SelfResult) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	if len(result.Warnings) > 0 {
		renderWarningsSection(renderer, "warnings", result.Warnings, false)
		renderer.Blank()
	}

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

	if len(result.Details) > 0 {
		renderer.Blank()
		renderer.Section("Details")
		renderer.Bullet("environment")
		renderTreeLines(renderer, result.Details, treeLineNormal)
	}
}

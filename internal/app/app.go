package app

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
	"github.com/tasuku43/gws/internal/config"
	"github.com/tasuku43/gws/internal/doctor"
	"github.com/tasuku43/gws/internal/gc"
	"github.com/tasuku43/gws/internal/gitcmd"
	"github.com/tasuku43/gws/internal/initcmd"
	"github.com/tasuku43/gws/internal/output"
	"github.com/tasuku43/gws/internal/paths"
	"github.com/tasuku43/gws/internal/repo"
	"github.com/tasuku43/gws/internal/repospec"
	"github.com/tasuku43/gws/internal/template"
	"github.com/tasuku43/gws/internal/ui"
	"github.com/tasuku43/gws/internal/workspace"
)

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
	case "gc":
		return runGC(ctx, rootDir, args[1:])
	case "repo":
		return runRepo(ctx, rootDir, args[1:])
	case "template":
		return runTemplate(ctx, rootDir, args[1:])
	case "new":
		return runWorkspaceNew(ctx, rootDir, args[1:], noPrompt)
	case "review":
		return runReview(ctx, rootDir, args[1:], noPrompt)
	case "add":
		return runWorkspaceAdd(ctx, rootDir, args[1:])
	case "ls":
		return runWorkspaceList(ctx, rootDir, args[1:])
	case "status":
		return runWorkspaceStatus(ctx, rootDir, args[1:])
	case "rm":
		return runWorkspaceRemove(ctx, rootDir, args[1:])
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

func runTemplate(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printTemplateHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "ls":
		return runTemplateList(ctx, rootDir, args[1:])
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

func runGC(ctx context.Context, rootDir string, args []string) error {
	gcFlags := flag.NewFlagSet("gc", flag.ContinueOnError)
	var dryRun bool
	var older string
	var helpFlag bool
	gcFlags.SetOutput(os.Stdout)
	gcFlags.Usage = func() {
		printGCHelp(os.Stdout)
	}
	gcFlags.BoolVar(&dryRun, "dry-run", false, "only list candidates")
	gcFlags.StringVar(&older, "older", "", "older than duration (e.g. 30d, 720h)")
	gcFlags.BoolVar(&helpFlag, "help", false, "show help")
	gcFlags.BoolVar(&helpFlag, "h", false, "show help")
	if err := gcFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printGCHelp(os.Stdout)
		return nil
	}
	if gcFlags.NArg() != 0 {
		return fmt.Errorf("usage: gws gc [--dry-run] [--older <duration>]")
	}

	olderThan, err := parseOlder(older)
	if err != nil {
		return err
	}

	opts := gc.Options{OlderThan: olderThan}
	now := time.Now().UTC()
	if dryRun {
		result, err := gc.DryRun(ctx, rootDir, opts, now)
		if err != nil {
			return err
		}
		writeGCText(result, true, older)
		return nil
	}

	result, err := gc.Run(ctx, rootDir, opts, now)
	if err != nil {
		return err
	}
	writeGCText(result, false, older)
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

	header := fmt.Sprintf("gws repo get (%s)", truncateMiddle(repoSpec, 80))
	renderer.Header(header)
	renderer.Blank()
	renderer.Section("Steps")
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
	prompted := false

	if templateName == "" || workspaceID == "" {
		if noPrompt {
			return fmt.Errorf("template name and workspace id are required without prompt")
		}
		prompted = true
		var err error
		templateName, workspaceID, err = promptTemplateAndID(rootDir, templateName, workspaceID, theme, useColor)
		if err != nil {
			return err
		}
	}

	cfg, err := config.Load(rootDir)
	if err != nil {
		return err
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

	header := "gws new"
	var headerParts []string
	if templateName != "" {
		headerParts = append(headerParts, fmt.Sprintf("template: %s", templateName))
	}
	if workspaceID != "" {
		headerParts = append(headerParts, fmt.Sprintf("workspace id: %s", workspaceID))
	}
	if len(headerParts) > 0 {
		header = fmt.Sprintf("%s (%s)", header, strings.Join(headerParts, ", "))
	}
	if !prompted {
		renderer.Header(header)
		renderer.Blank()
	} else {
		renderer.Blank()
	}
	renderer.Section("Steps")

	if len(missing) > 0 {
		if noPrompt {
			return fmt.Errorf("repo get required for: %s", strings.Join(missing, ", "))
		}
		output.Step(fmt.Sprintf("repo get required for %d repos", len(missing)))
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
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, filepath.Join(rootDir, "ws", workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID, cfg)
	if err != nil {
		return err
	}

	if err := applyTemplate(ctx, rootDir, workspaceID, tmpl, cfg); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("apply template failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _ := loadWorkspaceRepos(wsDir)
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
	cfg, err := config.Load(rootDir)
	if err != nil {
		return err
	}
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	prompted := false
	if workspaceID == "" || repoSpec == "" {
		prompted = true
		workspaces, wsWarn, err := workspace.List(rootDir)
		if err != nil {
			return err
		}
		if len(wsWarn) > 0 {
			// ignore warnings for selection
		}
		workspaceChoices := buildWorkspaceChoices(workspaces)
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
			value := repoSpecFromKey(entry.RepoKey, cfg)
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

	header := "gws add"
	var headerParts []string
	if workspaceID != "" {
		headerParts = append(headerParts, fmt.Sprintf("workspace id: %s", workspaceID))
	}
	if strings.TrimSpace(repoSpec) != "" {
		headerParts = append(headerParts, fmt.Sprintf("repo: %s", truncateMiddle(repoSpec, 80)))
	}
	if len(headerParts) > 0 {
		header = fmt.Sprintf("%s (%s)", header, strings.Join(headerParts, ", "))
	}
	if !prompted {
		renderer.Header(header)
		renderer.Blank()
	} else {
		renderer.Blank()
	}
	renderer.Section("Steps")
	output.Step(formatStep("worktree add", displayRepoName(repoSpec), worktreeDest(rootDir, workspaceID, repoSpec)))

	_, err = workspace.Add(ctx, rootDir, workspaceID, repoSpec, "", cfg, false)
	if err != nil {
		return err
	}
	wsDir := filepath.Join(rootDir, "ws", workspaceID)
	repos, _ := loadWorkspaceRepos(wsDir)
	renderer.Blank()
	renderer.Section("Result")
	renderWorkspaceBlock(renderer, workspaceID, repos)
	renderSuggestion(renderer, useColor, filepath.Join(rootDir, "ws", workspaceID))
	return nil
}

func runReview(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || (len(args) == 1 && isHelpArg(args[0])) {
		printReviewHelp(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gws review <PR URL>")
	}
	prURL := strings.TrimSpace(args[0])
	if prURL == "" {
		return fmt.Errorf("PR URL is required")
	}

	owner, repoName, number, err := parseGitHubPRURL(prURL)
	if err != nil {
		return err
	}

	cfg, err := config.Load(rootDir)
	if err != nil {
		return err
	}

	info, err := fetchGitHubPR(ctx, owner, repoName, number)
	if err != nil {
		return err
	}
	if info.HeadRepoFullName == "" || info.BaseRepoFullName == "" {
		return fmt.Errorf("failed to resolve PR repositories")
	}
	if info.HeadRepoFullName != info.BaseRepoFullName {
		return fmt.Errorf("fork PR is not supported")
	}
	repoURL := selectRepoURL(info, cfg)
	if repoURL == "" {
		return fmt.Errorf("cannot determine repo url for PR")
	}

	_, exists, err := repo.Exists(rootDir, repoURL)
	if err != nil {
		return err
	}
	workspaceID := fmt.Sprintf("REVIEW-PR-%d", info.Number)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	header := fmt.Sprintf("gws review (pr: %s, workspace id: %s)", truncateMiddle(prURL, 80), workspaceID)
	renderer.Header(header)
	renderer.Blank()
	renderer.Section("Steps")

	if !exists {
		if noPrompt {
			return fmt.Errorf("repo get required for: %s", repoURL)
		}
		output.Step("repo get required for 1 repo")
		output.Log(fmt.Sprintf("gws repo get %s", displayRepoSpec(repoURL)))
		confirm, err := ui.PromptConfirmInline("run now?", theme, useColor)
		if err != nil {
			return err
		}
		if !confirm {
			return fmt.Errorf("repo get required for: %s", repoURL)
		}
		output.Step(formatStep("repo get", displayRepoSpec(repoURL), repoDestForSpec(rootDir, repoURL)))
		if _, err := repo.Get(ctx, rootDir, repoURL); err != nil {
			return err
		}
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, filepath.Join(rootDir, "ws", workspaceID))))
	wsDir, err := workspace.New(ctx, rootDir, workspaceID, cfg)
	if err != nil {
		return err
	}

	store, err := repo.Open(ctx, rootDir, repoURL, false)
	if err != nil {
		return err
	}
	output.Step(formatStep("fetch PR head", info.HeadRefName, repoStoreRel(rootDir, repoURL)))
	if err := fetchPRHead(ctx, store.StorePath, info.HeadRefName); err != nil {
		return err
	}

	baseRef := fmt.Sprintf("refs/remotes/origin/%s", info.HeadRefName)
	output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
	if _, err := workspace.AddWithBranch(ctx, rootDir, workspaceID, repoURL, "", info.HeadRefName, baseRef, cfg, false); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("review failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _ := loadWorkspaceRepos(wsDir)
	renderWorkspaceBlock(renderer, workspaceID, repos)
	renderSuggestion(renderer, useColor, wsDir)
	return nil
}

type prInfo struct {
	Number           int
	HeadRefName      string
	HeadRepoFullName string
	HeadRepoSSHURL   string
	HeadRepoCloneURL string
	BaseRepoFullName string
}

func parseGitHubPRURL(raw string) (string, string, int, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR URL: %w", err)
	}
	if u.Hostname() != "github.com" {
		return "", "", 0, fmt.Errorf("only github.com PR URLs are supported")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return "", "", 0, fmt.Errorf("invalid PR URL path: %s", u.Path)
	}
	var owner, repo string
	var numStr string
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "pull" {
			if i < 2 {
				break
			}
			owner = parts[i-2]
			repo = parts[i-1]
			numStr = parts[i+1]
			break
		}
	}
	if owner == "" || repo == "" || numStr == "" {
		return "", "", 0, fmt.Errorf("invalid PR URL path: %s", u.Path)
	}
	number, err := strconv.Atoi(numStr)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number: %s", numStr)
	}
	return owner, repo, number, nil
}

func fetchGitHubPR(ctx context.Context, owner, repo string, number int) (prInfo, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, number)
	cmd := exec.CommandContext(ctx, "gh", "api", path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return prInfo{}, fmt.Errorf("gh api failed: %s", msg)
	}

	var payload struct {
		Number int `json:"number"`
		Head   struct {
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
				SSHURL   string `json:"ssh_url"`
				CloneURL string `json:"clone_url"`
			} `json:"repo"`
		} `json:"head"`
		Base struct {
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"base"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		return prInfo{}, fmt.Errorf("parse gh response: %w", err)
	}
	info := prInfo{
		Number:           payload.Number,
		HeadRefName:      payload.Head.Ref,
		HeadRepoFullName: payload.Head.Repo.FullName,
		HeadRepoSSHURL:   payload.Head.Repo.SSHURL,
		HeadRepoCloneURL: payload.Head.Repo.CloneURL,
		BaseRepoFullName: payload.Base.Repo.FullName,
	}
	if info.Number == 0 {
		info.Number = number
	}
	return info, nil
}

func selectRepoURL(info prInfo, cfg config.Config) string {
	switch strings.ToLower(strings.TrimSpace(cfg.Repo.DefaultProtocol)) {
	case "ssh":
		if info.HeadRepoSSHURL != "" {
			return info.HeadRepoSSHURL
		}
	case "https":
		if info.HeadRepoCloneURL != "" {
			return info.HeadRepoCloneURL
		}
	}
	if info.HeadRepoSSHURL != "" {
		return info.HeadRepoSSHURL
	}
	return info.HeadRepoCloneURL
}

func fetchPRHead(ctx context.Context, storePath, headRef string) error {
	if strings.TrimSpace(headRef) == "" {
		return fmt.Errorf("PR head ref is empty")
	}
	gitcmd.Logf("git fetch origin %s", headRef)
	if _, err := gitcmd.Run(ctx, []string{"fetch", "origin", headRef}, gitcmd.Options{Dir: storePath}); err != nil {
		return err
	}
	return nil
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

func renderWorkspaceRepos(r *ui.Renderer, repos []workspace.Repo, extraIndent string) {
	for i, repo := range repos {
		prefix := "├─ "
		if i == len(repos)-1 {
			prefix = "└─ "
		}
		name := repo.Alias
		if strings.TrimSpace(name) == "" {
			name = repo.RepoKey
		}
		if r != nil {
			r.TreeLineBranchMuted(extraIndent+prefix, name, repo.Branch)
			continue
		}
		line := fmt.Sprintf("%s%s%s%s", output.Indent, extraIndent, prefix, name)
		if strings.TrimSpace(repo.Branch) != "" {
			line += fmt.Sprintf(" (branch: %s)", repo.Branch)
		}
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

func loadWorkspaceRepos(wsDir string) ([]workspace.Repo, error) {
	manifestPath := filepath.Join(wsDir, ".gws", "manifest.yaml")
	manifest, err := workspace.LoadManifest(manifestPath)
	if err == nil {
		return manifest.Repos, nil
	}
	entries, err := os.ReadDir(wsDir)
	if err != nil {
		return nil, fmt.Errorf("read workspace dir: %w", err)
	}
	var repos []workspace.Repo
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".gws" {
			continue
		}
		repos = append(repos, workspace.Repo{Alias: entry.Name()})
	}
	return repos, nil
}

func buildWorkspaceChoices(entries []workspace.Entry) []ui.WorkspaceChoice {
	var choices []ui.WorkspaceChoice
	for _, entry := range entries {
		choices = append(choices, buildWorkspaceChoice(entry))
	}
	return choices
}

func buildWorkspaceChoice(entry workspace.Entry) ui.WorkspaceChoice {
	choice := ui.WorkspaceChoice{ID: entry.WorkspaceID}
	if entry.Manifest == nil {
		return choice
	}
	for _, repoEntry := range entry.Manifest.Repos {
		name := repoEntry.Alias
		if strings.TrimSpace(name) == "" {
			name = repoEntry.RepoKey
		}
		label := name
		if strings.TrimSpace(repoEntry.Branch) != "" {
			label = fmt.Sprintf("%s (branch: %s)", name, repoEntry.Branch)
		}
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
	wsPath := filepath.Join(rootDir, "ws", workspaceID, spec.Repo)
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

func renderSuggestion(r *ui.Renderer, useColor bool, path string) {
	if !useColor {
		return
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	r.Blank()
	r.Section("Suggestion")
	r.Bullet(fmt.Sprintf("cd %s", path))
}

func repoSpecFromKey(repoKey string, cfg config.Config) string {
	trimmed := strings.TrimSuffix(repoKey, ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return repoKey
	}
	host := parts[0]
	owner := parts[1]
	repo := parts[2]
	if strings.EqualFold(strings.TrimSpace(cfg.Repo.DefaultProtocol), "ssh") {
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

func classifyWorkspaceRemoval(ctx context.Context, rootDir string, entries []workspace.Entry) ([]ui.WorkspaceChoice, []ui.BlockedChoice) {
	var removable []ui.WorkspaceChoice
	var blocked []ui.BlockedChoice
	for _, entry := range entries {
		reason := workspaceRemoveReason(ctx, rootDir, entry)
		if strings.TrimSpace(reason) == "" {
			removable = append(removable, buildWorkspaceChoice(entry))
			continue
		}
		blocked = append(blocked, ui.BlockedChoice{
			Label: fmt.Sprintf("%s (%s)", entry.WorkspaceID, reason),
		})
	}
	return removable, blocked
}

func workspaceRemoveReason(ctx context.Context, rootDir string, entry workspace.Entry) string {
	if entry.Warning != nil {
		return fmt.Sprintf("manifest: %s", compactError(entry.Warning))
	}
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

func applyTemplate(ctx context.Context, rootDir, workspaceID string, tmpl template.Template, cfg config.Config) error {
	total := len(tmpl.Repos)
	for i, repoSpec := range tmpl.Repos {
		output.Step(formatStepWithIndex("worktree add", displayRepoName(repoSpec), worktreeDest(rootDir, workspaceID, repoSpec), i+1, total))
		if _, err := workspace.Add(ctx, rootDir, workspaceID, repoSpec, "", cfg, true); err != nil {
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
	writeWorkspaceListText(entries, warnings)
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
	showHeader := true
	if workspaceID == "" {
		workspaces, wsWarn, err := workspace.List(rootDir)
		if err != nil {
			return err
		}
		if len(wsWarn) > 0 {
			// ignore warnings for selection
		}
		workspaceChoices := buildWorkspaceChoices(workspaces)
		if len(workspaceChoices) == 0 {
			return fmt.Errorf("no workspaces found")
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		workspaceID, err = ui.PromptWorkspace("gws status", workspaceChoices, theme, useColor)
		if err != nil {
			return err
		}
		showHeader = false
	}
	result, err := workspace.Status(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}

	writeWorkspaceStatusText(result, showHeader)
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

	showHeader := true
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
			renderer.Header("gws rm")
			renderer.Blank()
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
		showHeader = false
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	header := "gws rm"
	if strings.TrimSpace(workspaceID) != "" {
		header = fmt.Sprintf("%s (workspace id: %s)", header, workspaceID)
	}
	if showHeader {
		renderer.Header(header)
		renderer.Blank()
	} else {
		renderer.Blank()
	}
	renderer.Section("Steps")
	output.Step(formatStep("remove workspace", workspaceID, relPath(rootDir, filepath.Join(rootDir, "ws", workspaceID))))

	if err := workspace.Remove(ctx, rootDir, workspaceID); err != nil {
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	renderer.Bullet(fmt.Sprintf("%s removed", workspaceID))
	return nil
}

func writeWorkspaceStatusText(result workspace.StatusResult, showHeader bool) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	header := "gws status"
	if strings.TrimSpace(result.WorkspaceID) != "" {
		header = fmt.Sprintf("%s (workspace id: %s)", header, result.WorkspaceID)
	}
	if showHeader {
		renderer.Header(header)
		renderer.Blank()
	} else {
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
		if repo.Error != nil {
			renderer.Warn(fmt.Sprintf("warning: %s: %v", repo.Alias, repo.Error))
		}
	}
}

func writeWorkspaceListText(entries []workspace.Entry, warnings []error) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Header("gws ls")
	renderer.Blank()
	renderer.Section("Result")
	for _, entry := range entries {
		var repos []workspace.Repo
		if entry.Manifest != nil {
			repos = entry.Manifest.Repos
		}

		renderWorkspaceBlock(renderer, entry.WorkspaceID, repos)

		if entry.Warning != nil {
			renderer.Bullet(fmt.Sprintf("%s warning", entry.WorkspaceID))
			renderTreeLines(renderer, []string{entry.Warning.Error()}, treeLineWarn)
		}
	}
	if len(warnings) > 0 {
		renderer.Blank()
		renderer.Section("Info")
		renderer.Bullet("warnings")
		var lines []string
		for _, warning := range warnings {
			lines = append(lines, warning.Error())
		}
		renderTreeLines(renderer, lines, treeLineWarn)
	}
}

func writeRepoListText(entries []repo.Entry, warnings []error) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Header("gws repo ls")
	renderer.Blank()
	renderer.Section("Result")
	for _, entry := range entries {
		renderer.Result(fmt.Sprintf("%s\t%s", entry.RepoKey, entry.StorePath))
	}
	if len(warnings) > 0 {
		renderer.Blank()
		renderer.Section("Info")
		renderer.Bullet("warnings")
		var lines []string
		for _, warning := range warnings {
			lines = append(lines, warning.Error())
		}
		renderTreeLines(renderer, lines, treeLineWarn)
	}
}

func writeTemplateListText(file template.File, names []string) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Header("gws template ls")
	renderer.Blank()
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

	renderer.Header("gws init")
	renderer.Blank()
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
}
func writeGCText(result gc.Result, dryRun bool, older string) {
	action := "gc"
	if dryRun {
		action = "gc --dry-run"
	}
	if strings.TrimSpace(older) != "" {
		fmt.Fprintf(os.Stdout, "%s (older=%s)\n", action, older)
	}
	fmt.Fprintln(os.Stdout, "id\tlast_used_at\treason\tpath")
	for _, candidate := range result.Candidates {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%s\n", candidate.WorkspaceID, candidate.LastUsedAt, candidate.Reason, candidate.WorkspacePath)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
}

func parseOlder(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	if strings.HasSuffix(trimmed, "d") {
		raw := strings.TrimSuffix(trimmed, "d")
		days, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("invalid --older value: %s", value)
		}
		if days < 0 {
			return 0, fmt.Errorf("invalid --older value: %s", value)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	parsed, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid --older value: %s", value)
	}
	return parsed, nil
}
func writeDoctorText(result doctor.Result, fixed []string) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Header("gws doctor")
	renderer.Blank()
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

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
	var jsonFlag bool
	var noPrompt bool
	verboseFlag := envBool("GWS_VERBOSE")
	var helpFlag bool
	fs.StringVar(&rootFlag, "root", "", "override gws root")
	fs.BoolVar(&jsonFlag, "json", false, "machine readable output")
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
		return runInit(rootDir, jsonFlag, args[1:])
	case "doctor":
		return runDoctor(ctx, rootDir, jsonFlag, args[1:])
	case "gc":
		return runGC(ctx, rootDir, jsonFlag, args[1:])
	case "repo":
		return runRepo(ctx, rootDir, jsonFlag, args[1:])
	case "template":
		return runTemplate(ctx, rootDir, jsonFlag, args[1:])
	case "new":
		return runWorkspaceNew(ctx, rootDir, args[1:], noPrompt)
	case "review":
		return runReview(ctx, rootDir, args[1:], noPrompt)
	case "add":
		return runWorkspaceAdd(ctx, rootDir, args[1:])
	case "ls":
		return runWorkspaceList(ctx, rootDir, jsonFlag, args[1:])
	case "status":
		return runWorkspaceStatus(ctx, rootDir, jsonFlag, args[1:])
	case "rm":
		return runWorkspaceRemove(ctx, rootDir, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runInit(rootDir string, jsonFlag bool, args []string) error {
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
	if jsonFlag {
		return writeInitJSON(result)
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

func runTemplate(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printTemplateHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "ls":
		return runTemplateList(ctx, rootDir, jsonFlag, args[1:])
	default:
		return fmt.Errorf("unknown template subcommand: %s", args[0])
	}
}

func runTemplateList(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
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
	if jsonFlag {
		return writeTemplateListJSON(names)
	}
	writeTemplateListText(names)
	return nil
}

func runDoctor(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
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
		if jsonFlag {
			return writeDoctorJSON(result.Result, result.Fixed)
		}
		writeDoctorText(result.Result, result.Fixed)
		return nil
	}

	result, err := doctor.Check(ctx, rootDir, now)
	if err != nil {
		return err
	}
	if jsonFlag {
		return writeDoctorJSON(result, nil)
	}
	writeDoctorText(result, nil)
	return nil
}

func runGC(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
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
		if jsonFlag {
			return writeGCJSON(result, true, older)
		}
		writeGCText(result, true, older)
		return nil
	}

	result, err := gc.Run(ctx, rootDir, opts, now)
	if err != nil {
		return err
	}
	if jsonFlag {
		return writeGCJSON(result, false, older)
	}
	writeGCText(result, false, older)
	return nil
}

func runRepo(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printRepoHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "get":
		return runRepoGet(ctx, rootDir, args[1:])
	case "ls":
		return runRepoList(ctx, rootDir, jsonFlag, args[1:])
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

	header := fmt.Sprintf("gws repo get (%s)", repoSpec)
	renderer.Header(header)
	renderer.Blank()
	renderer.Section("Steps")
	output.Step(fmt.Sprintf("repo get %s", repoSpec))

	store, err := repo.Get(ctx, rootDir, repoSpec)
	if err != nil {
		return err
	}
	renderer.Blank()
	renderer.Section("Result")
	renderer.Result(fmt.Sprintf("%s\t%s", store.RepoKey, store.StorePath))
	return nil
}

func runRepoList(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
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
	if jsonFlag {
		return writeRepoListJSON(entries, warnings)
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
			output.Log(fmt.Sprintf("gws repo get %s", repoSpec))
		}
		confirm, err := ui.PromptConfirmInline("run now?", theme, useColor)
		if err != nil {
			return err
		}
		if !confirm {
			return fmt.Errorf("repo get required for: %s", strings.Join(missing, ", "))
		}
		for _, repoSpec := range missing {
			output.Step(fmt.Sprintf("repo get %s", repoSpec))
			if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
				return err
			}
		}
	}

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
		var workspaceChoices []ui.WorkspaceChoice
		for _, entry := range workspaces {
			choice := ui.WorkspaceChoice{ID: entry.WorkspaceID}
			if entry.Manifest != nil {
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
			}
			workspaceChoices = append(workspaceChoices, choice)
		}
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
		headerParts = append(headerParts, fmt.Sprintf("repo: %s", repoSpec))
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

	display := repoSpec
	if spec, err := repospec.Normalize(repoSpec); err == nil && spec.Repo != "" {
		display = spec.Repo
	}
	output.Step(fmt.Sprintf("worktree add %s", display))

	_, err = workspace.Add(ctx, rootDir, workspaceID, repoSpec, "", cfg, false)
	if err != nil {
		return err
	}
	wsDir := filepath.Join(rootDir, "ws", workspaceID)
	repos, _ := loadWorkspaceRepos(wsDir)
	renderer.Blank()
	renderer.Section("Result")
	renderWorkspaceBlock(renderer, workspaceID, repos)
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

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	_, exists, err := repo.Exists(rootDir, repoURL)
	if err != nil {
		return err
	}
	if !exists {
		if noPrompt {
			return fmt.Errorf("repo get required for: %s", repoURL)
		}
		fmt.Fprintf(os.Stdout, "%srepo get required for 1 repo.\n", output.Indent)
		printRepoGetCommands([]string{repoURL})
		confirm, err := ui.PromptConfirmInline("run now?", theme, useColor)
		if err != nil {
			return err
		}
		if !confirm {
			return fmt.Errorf("repo get required for: %s", repoURL)
		}
		if _, err := repo.Get(ctx, rootDir, repoURL); err != nil {
			return err
		}
	}

	workspaceID := fmt.Sprintf("REVIEW-PR-%d", info.Number)
	wsDir, err := workspace.New(ctx, rootDir, workspaceID, cfg)
	if err != nil {
		return err
	}

	store, err := repo.Open(ctx, rootDir, repoURL, false)
	if err != nil {
		return err
	}
	if err := fetchPRHead(ctx, store.StorePath, info.HeadRefName); err != nil {
		return err
	}

	baseRef := fmt.Sprintf("refs/remotes/origin/%s", info.HeadRefName)
	if _, err := workspace.AddWithBranch(ctx, rootDir, workspaceID, repoURL, "", info.HeadRefName, baseRef, cfg, false); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("review failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "%s\x1b[32mWorkspace ready!\x1b[0m\n\n", output.Indent)
	if err := printWorkspaceTree(wsDir, nil); err != nil {
		return err
	}
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

func printWorkspaceTree(wsDir string, r *ui.Renderer) error {
	repos, err := loadWorkspaceRepos(wsDir)
	if err != nil {
		return err
	}
	if r == nil {
		fmt.Fprintf(os.Stdout, "%s%s\n", output.Indent, wsDir)
	}
	renderWorkspaceRepos(r, repos, "")
	return nil
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

func displayRepoKey(repoKey string) string {
	display := strings.TrimSuffix(repoKey, ".git")
	display = strings.TrimSuffix(display, "/")
	return display
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
	for _, repoSpec := range tmpl.Repos {
		display := repoSpec
		if spec, err := repospec.Normalize(repoSpec); err == nil {
			display = spec.Repo
		}
		output.Step(fmt.Sprintf("worktree add %s", display))
		if _, err := workspace.Add(ctx, rootDir, workspaceID, repoSpec, "", cfg, true); err != nil {
			return err
		}
	}
	return nil
}

func runWorkspaceList(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
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
	if jsonFlag {
		return writeWorkspaceListJSON(entries, warnings)
	}
	writeWorkspaceListText(entries, warnings)
	return nil
}

func runWorkspaceStatus(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printStatusHelp(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gws status <WORKSPACE_ID>")
	}
	workspaceID := args[0]
	result, err := workspace.Status(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}

	if jsonFlag {
		return writeWorkspaceStatusJSON(result)
	}
	writeWorkspaceStatusText(result)
	return nil
}

func runWorkspaceRemove(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printRmHelp(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gws rm <WORKSPACE_ID>")
	}
	return workspace.Remove(ctx, rootDir, args[0])
}

type workspaceStatusJSON struct {
	SchemaVersion int                       `json:"schema_version"`
	Command       string                    `json:"command"`
	WorkspaceID   string                    `json:"workspace_id"`
	Repos         []workspaceStatusRepoJSON `json:"repos"`
}

type workspaceStatusRepoJSON struct {
	Alias          string `json:"alias"`
	Branch         string `json:"branch"`
	Head           string `json:"head,omitempty"`
	Dirty          bool   `json:"dirty"`
	UntrackedCount int    `json:"untracked_count"`
	Error          string `json:"error,omitempty"`
}

func writeWorkspaceStatusJSON(result workspace.StatusResult) error {
	out := workspaceStatusJSON{
		SchemaVersion: 1,
		Command:       "status",
		WorkspaceID:   result.WorkspaceID,
	}
	for _, repo := range result.Repos {
		repoOut := workspaceStatusRepoJSON{
			Alias:          repo.Alias,
			Branch:         repo.Branch,
			Head:           repo.Head,
			Dirty:          repo.Dirty,
			UntrackedCount: repo.UntrackedCount,
		}
		if repo.Error != nil {
			repoOut.Error = repo.Error.Error()
		}
		out.Repos = append(out.Repos, repoOut)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeWorkspaceStatusText(result workspace.StatusResult) {
	fmt.Fprintln(os.Stdout, "alias\tbranch\thead\tdirty\tuntracked")
	for _, repo := range result.Repos {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%t\t%d\n", repo.Alias, repo.Branch, repo.Head, repo.Dirty, repo.UntrackedCount)
		if repo.Error != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", repo.Alias, repo.Error)
		}
	}
}

type workspaceListJSON struct {
	SchemaVersion int                      `json:"schema_version"`
	Command       string                   `json:"command"`
	Workspaces    []workspaceListEntryJSON `json:"workspaces"`
}

type workspaceListEntryJSON struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspacePath string `json:"workspace_path"`
	ManifestPath  string `json:"manifest_path"`
	RepoCount     int    `json:"repo_count"`
	Warning       string `json:"warning,omitempty"`
}

func writeWorkspaceListJSON(entries []workspace.Entry, warnings []error) error {
	out := workspaceListJSON{
		SchemaVersion: 1,
		Command:       "ls",
	}
	for _, entry := range entries {
		repoCount := 0
		if entry.Manifest != nil {
			repoCount = len(entry.Manifest.Repos)
		}
		item := workspaceListEntryJSON{
			WorkspaceID:   entry.WorkspaceID,
			WorkspacePath: entry.WorkspacePath,
			ManifestPath:  entry.ManifestPath,
			RepoCount:     repoCount,
		}
		if entry.Warning != nil {
			item.Warning = entry.Warning.Error()
		}
		out.Workspaces = append(out.Workspaces, item)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	return nil
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
			renderer.Warn(fmt.Sprintf("warning: %s: %v", entry.WorkspaceID, entry.Warning))
		}
	}
	for _, warning := range warnings {
		renderer.Warn(fmt.Sprintf("warning: %v", warning))
	}
}

type repoListJSON struct {
	SchemaVersion int                 `json:"schema_version"`
	Command       string              `json:"command"`
	Repos         []repoListEntryJSON `json:"repos"`
}

type repoListEntryJSON struct {
	RepoKey   string `json:"repo_key"`
	StorePath string `json:"store_path"`
	Warning   string `json:"warning,omitempty"`
}

func writeRepoListJSON(entries []repo.Entry, warnings []error) error {
	out := repoListJSON{
		SchemaVersion: 1,
		Command:       "repo.ls",
	}
	for _, entry := range entries {
		item := repoListEntryJSON{
			RepoKey:   entry.RepoKey,
			StorePath: entry.StorePath,
		}
		out.Repos = append(out.Repos, item)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	return nil
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
	for _, warning := range warnings {
		renderer.Warn(fmt.Sprintf("warning: %v", warning))
	}
}

type templateListJSON struct {
	SchemaVersion int      `json:"schema_version"`
	Command       string   `json:"command"`
	Templates     []string `json:"templates"`
}

type templateShowJSON struct {
	SchemaVersion int      `json:"schema_version"`
	Command       string   `json:"command"`
	Name          string   `json:"name"`
	Repos         []string `json:"repos"`
}

type templateRemoveJSON struct {
	SchemaVersion int    `json:"schema_version"`
	Command       string `json:"command"`
	Name          string `json:"name"`
	Removed       bool   `json:"removed"`
}

func writeTemplateListJSON(names []string) error {
	out := templateListJSON{
		SchemaVersion: 1,
		Command:       "template.ls",
		Templates:     names,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeTemplateListText(names []string) {
	for _, name := range names {
		fmt.Fprintln(os.Stdout, name)
	}
}

func writeTemplateShowJSON(name string, tmpl template.Template) error {
	out := templateShowJSON{
		SchemaVersion: 1,
		Command:       "template.show",
		Name:          name,
		Repos:         tmpl.Repos,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeTemplateShowText(name string, tmpl template.Template) {
	fmt.Fprintf(os.Stdout, "%s\n", name)
	for _, repo := range tmpl.Repos {
		fmt.Fprintf(os.Stdout, " - %s\n", repo)
	}
}

func writeTemplateRemoveJSON(name string) error {
	out := templateRemoveJSON{
		SchemaVersion: 1,
		Command:       "template.rm",
		Name:          name,
		Removed:       true,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

type initJSON struct {
	SchemaVersion int      `json:"schema_version"`
	Command       string   `json:"command"`
	RootDir       string   `json:"root_dir"`
	CreatedDirs   []string `json:"created_dirs"`
	CreatedFiles  []string `json:"created_files"`
	SkippedDirs   []string `json:"skipped_dirs"`
	SkippedFiles  []string `json:"skipped_files"`
}

func writeInitJSON(result initcmd.Result) error {
	out := initJSON{
		SchemaVersion: 1,
		Command:       "init",
		RootDir:       result.RootDir,
		CreatedDirs:   result.CreatedDirs,
		CreatedFiles:  result.CreatedFiles,
		SkippedDirs:   result.SkippedDirs,
		SkippedFiles:  result.SkippedFiles,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeInitText(result initcmd.Result) {
	fmt.Fprintf(os.Stdout, "root: %s\n", result.RootDir)
	for _, dir := range result.CreatedDirs {
		fmt.Fprintf(os.Stdout, "created\t%s\n", dir)
	}
	for _, file := range result.CreatedFiles {
		fmt.Fprintf(os.Stdout, "created\t%s\n", file)
	}
	for _, dir := range result.SkippedDirs {
		fmt.Fprintf(os.Stdout, "exists\t%s\n", dir)
	}
	for _, file := range result.SkippedFiles {
		fmt.Fprintf(os.Stdout, "exists\t%s\n", file)
	}
}

type gcJSON struct {
	SchemaVersion int               `json:"schema_version"`
	Command       string            `json:"command"`
	DryRun        bool              `json:"dry_run"`
	Older         string            `json:"older,omitempty"`
	Candidates    []gcCandidateJSON `json:"candidates"`
}

type gcCandidateJSON struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspacePath string `json:"workspace_path"`
	LastUsedAt    string `json:"last_used_at"`
	Reason        string `json:"reason"`
}

func writeGCJSON(result gc.Result, dryRun bool, older string) error {
	out := gcJSON{
		SchemaVersion: 1,
		Command:       "gc",
		DryRun:        dryRun,
		Older:         strings.TrimSpace(older),
	}
	for _, candidate := range result.Candidates {
		out.Candidates = append(out.Candidates, gcCandidateJSON{
			WorkspaceID:   candidate.WorkspaceID,
			WorkspacePath: candidate.WorkspacePath,
			LastUsedAt:    candidate.LastUsedAt,
			Reason:        candidate.Reason,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	return nil
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

type doctorJSON struct {
	SchemaVersion int               `json:"schema_version"`
	Command       string            `json:"command"`
	Issues        []doctorIssueJSON `json:"issues"`
	Fixed         []string          `json:"fixed,omitempty"`
}

type doctorIssueJSON struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

func writeDoctorJSON(result doctor.Result, fixed []string) error {
	out := doctorJSON{
		SchemaVersion: 1,
		Command:       "doctor",
	}
	for _, issue := range result.Issues {
		out.Issues = append(out.Issues, doctorIssueJSON{
			Kind:    issue.Kind,
			Path:    issue.Path,
			Message: issue.Message,
		})
	}
	if len(fixed) > 0 {
		out.Fixed = fixed
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	return nil
}

func writeDoctorText(result doctor.Result, fixed []string) {
	fmt.Fprintln(os.Stdout, "kind\tpath\tmessage")
	for _, issue := range result.Issues {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\n", issue.Kind, issue.Path, issue.Message)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	if len(fixed) > 0 {
		for _, path := range fixed {
			fmt.Fprintf(os.Stdout, "fixed\t%s\n", path)
		}
	}
}

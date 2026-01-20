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
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/create"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/debuglog"
	"github.com/tasuku43/gwst/internal/infra/gitcmd"
	"github.com/tasuku43/gwst/internal/infra/output"
	"github.com/tasuku43/gwst/internal/infra/prefetcher"
	"github.com/tasuku43/gwst/internal/ui"
)

func runCreateIssue(ctx context.Context, rootDir, issueURL, workspaceID, branch, baseRef string, noPrompt bool, prefetch *prefetcher.Prefetcher) error {
	prefetch = prefetcher.Ensure(prefetch, defaultPrefetchTimeout)
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
		return runIssuePicker(ctx, rootDir, noPrompt, "gwst create", prefetch)
	}

	req, err := parseIssueURL(issueURL)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(req.Provider), "github") {
		return fmt.Errorf("unsupported issue provider: %s", req.Provider)
	}
	repoURL := buildRepoURLFromParts(req.Host, req.Owner, req.Repo)
	description := ""
	if strings.EqualFold(strings.TrimSpace(req.Provider), "github") {
		provider, err := providerByName(req.Provider)
		if err != nil {
			return err
		}
		issue, err := provider.FetchIssue(ctx, req.Host, req.Owner, req.Repo, req.Number)
		if err != nil {
			return err
		}
		description = issue.Title
	}
	prefetchStarted := false
	if _, exists, err := repo.Exists(rootDir, repoURL); err != nil {
		return err
	} else if exists {
		started, err := prefetch.Start(ctx, rootDir, repoURL)
		if err != nil {
			return err
		}
		prefetchStarted = started
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
	if prefetchStarted {
		renderTreeLines(renderer, []string{"prefetch: git fetch origin (background)"}, treeLineNormal)
	}
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
		if _, err := prefetch.Start(ctx, rootDir, repoURL); err != nil {
			return err
		}
	}
	store, err := repo.Open(ctx, rootDir, repoURL, false)
	if err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := create.CreateWorkspace(ctx, rootDir, workspaceID, workspace.Metadata{
		Description: description,
		Mode:        workspace.MetadataModeIssue,
		SourceURL:   issueURL,
	})
	if err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return create.FailWorkspaceMetadata(err, rollbackErr)
		}
		return err
	}

	if err := prefetch.Wait(ctx, repoURL); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("prefetch failed: %w (rollback failed: %v)", err, rollbackErr)
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
	if err := rebuildManifest(ctx, rootDir); err != nil {
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestions(renderer, useColor, []string{
		"gwst open",
	})
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
	prefetch := prefetcher.New(defaultPrefetchTimeout)

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
		return runIssuePicker(ctx, rootDir, noPrompt, "gwst create", prefetch)
	}

	if issueFlags.NArg() != 1 {
		return fmt.Errorf("usage: gwst create --issue [<ISSUE_URL>] [--workspace-id <id>] [--branch <name>] [--base <ref>]")
	}

	raw := strings.TrimSpace(issueFlags.Arg(0))
	if raw == "" {
		return fmt.Errorf("issue URL is required")
	}

	req, err := parseIssueURL(raw)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(req.Provider), "github") {
		return fmt.Errorf("unsupported issue provider: %s", req.Provider)
	}
	repoURL := buildRepoURLFromParts(req.Host, req.Owner, req.Repo)
	description := ""
	if strings.EqualFold(strings.TrimSpace(req.Provider), "github") {
		provider, err := providerByName(req.Provider)
		if err != nil {
			return err
		}
		issue, err := provider.FetchIssue(ctx, req.Host, req.Owner, req.Repo, req.Number)
		if err != nil {
			return err
		}
		description = issue.Title
	}
	prefetchStarted := false
	if _, exists, err := repo.Exists(rootDir, repoURL); err != nil {
		return err
	} else if exists {
		started, err := prefetch.Start(ctx, rootDir, repoURL)
		if err != nil {
			return err
		}
		prefetchStarted = started
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
	if prefetchStarted {
		renderTreeLines(renderer, []string{"prefetch: git fetch origin (background)"}, treeLineNormal)
	}
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
		if _, err := prefetch.Start(ctx, rootDir, repoURL); err != nil {
			return err
		}
	}
	store, err := repo.Open(ctx, rootDir, repoURL, false)
	if err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := create.CreateWorkspace(ctx, rootDir, workspaceID, workspace.Metadata{
		Description: description,
		Mode:        workspace.MetadataModeIssue,
		SourceURL:   raw,
	})
	if err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return create.FailWorkspaceMetadata(err, rollbackErr)
		}
		return err
	}

	if err := prefetch.Wait(ctx, repoURL); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("prefetch failed: %w (rollback failed: %v)", err, rollbackErr)
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
	if err := rebuildManifest(ctx, rootDir); err != nil {
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestions(renderer, useColor, []string{
		"gwst open",
	})
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

func runIssuePicker(ctx context.Context, rootDir string, noPrompt bool, title string, prefetch *prefetcher.Prefetcher) error {
	prefetch = prefetcher.Ensure(prefetch, defaultPrefetchTimeout)
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	repoChoices, err := buildIssueRepoChoices(rootDir)
	if err != nil {
		return err
	}
	if len(repoChoices) == 0 {
		return fmt.Errorf("no repos with supported hosts found")
	}

	promptChoices, repoByValue := toIssuePromptChoices(repoChoices)
	loadIssue := func(value string) ([]ui.PromptChoice, error) {
		selected, ok := repoByValue[value]
		if !ok {
			return nil, fmt.Errorf("selected repo not found")
		}
		if strings.ToLower(selected.Provider) != "github" {
			return nil, fmt.Errorf("issue picker supports GitHub only for now: %s", selected.Host)
		}
		provider, err := providerByName(selected.Provider)
		if err != nil {
			return nil, err
		}
		issues, err := provider.FetchIssues(ctx, selected.Host, selected.Owner, selected.Repo)
		if err != nil {
			return nil, err
		}
		return buildIssueChoices(issues), nil
	}
	validateBranch := func(value string) error {
		return workspace.ValidateBranchName(ctx, value)
	}
	onReposResolved := func(repos []string) {
		for _, repoSpec := range repos {
			_, _ = prefetch.Start(ctx, rootDir, repoSpec)
		}
	}
	mode, _, _, _, _, _, _, issueRepo, issueSelections, _, err := ui.PromptCreateFlow(title, "issue", "", "", nil, nil, nil, nil, nil, promptChoices, nil, loadIssue, nil, onReposResolved, validateBranch, theme, useColor, "")
	if err != nil {
		return err
	}
	if mode != "issue" {
		return fmt.Errorf("unknown mode: %s", mode)
	}
	return runCreateIssueSelected(ctx, rootDir, noPrompt, issueRepo, issueSelections, prefetch)
}

func runCreateIssueSelected(ctx context.Context, rootDir string, noPrompt bool, repoSpec string, selectedIssues []ui.IssueSelection, prefetch *prefetcher.Prefetcher) error {
	prefetch = prefetcher.Ensure(prefetch, defaultPrefetchTimeout)
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

	type issueSelectionPlan struct {
		number int
		branch string
		label  string
		raw    ui.IssueSelection
	}
	plans := make([]issueSelectionPlan, 0, len(selectedIssues))
	branchSet := make(map[string]struct{}, len(selectedIssues))
	for _, selection := range selectedIssues {
		num, err := strconv.Atoi(strings.TrimSpace(selection.Value))
		if err != nil {
			return fmt.Errorf("invalid issue number: %s", selection.Value)
		}
		branch := strings.TrimSpace(selection.Branch)
		if branch == "" {
			branch = fmt.Sprintf("issue/%d", num)
		}
		plans = append(plans, issueSelectionPlan{
			number: num,
			branch: branch,
			label:  selection.Label,
			raw:    selection,
		})
		if branch != "" {
			branchSet[branch] = struct{}{}
		}
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
	if _, err := prefetch.Start(ctx, rootDir, repoSpec); err != nil {
		return err
	}
	store, err := repo.Open(ctx, rootDir, repoSpec, false)
	if err != nil {
		return err
	}
	if err := prefetch.Wait(ctx, repoSpec); err != nil {
		return err
	}

	branches := make([]string, 0, len(branchSet))
	for branch := range branchSet {
		branches = append(branches, branch)
	}
	remoteBranches, err := localRemoteBranches(ctx, store.StorePath, branches)
	if err != nil {
		return err
	}
	baseRef, err := workspace.ResolveBaseRef(ctx, store.StorePath)
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

	for _, plan := range plans {
		num := plan.number
		description := issueTitleFromLabel(plan.label, num)
		workspaceID := formatIssueWorkspaceID(selectedRepo.Owner, selectedRepo.Repo, num)
		branch := plan.branch
		if err := workspace.ValidateBranchName(ctx, branch); err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
		sourceURL := buildIssueURLFromParts(selectedRepo.Host, selectedRepo.Owner, selectedRepo.Repo, num)
		wsDir, err := create.CreateWorkspace(ctx, rootDir, workspaceID, workspace.Metadata{
			Description: description,
			Mode:        workspace.MetadataModeIssue,
			SourceURL:   sourceURL,
		})
		if err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = create.FailWorkspaceMetadata(err, rollbackErr)
			} else {
				failure = err
			}
			failureID = workspaceID
			break
		}

		output.Step(formatStep("worktree add", displayRepoName(repoURL), worktreeDest(rootDir, workspaceID, repoURL)))
		if _, err := addIssueWorktreeWithRemoteInfo(ctx, rootDir, workspaceID, repoURL, branch, baseRef, store.StorePath, remoteBranches[branch]); err != nil {
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
		if err := rebuildManifest(ctx, rootDir); err != nil {
			return err
		}
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

func localRemoteBranches(ctx context.Context, storePath string, branches []string) (map[string]bool, error) {
	remote := make(map[string]bool, len(branches))
	if strings.TrimSpace(storePath) == "" || len(branches) == 0 {
		return remote, nil
	}
	for _, branch := range branches {
		if strings.TrimSpace(branch) == "" {
			continue
		}
		exists, err := remoteTrackingExists(ctx, storePath, branch)
		if err != nil {
			return nil, err
		}
		if exists {
			remote[branch] = true
		}
	}
	return remote, nil
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
		if !isGitHubHost(host) {
			continue
		}
		owner := parts[1]
		repoName := parts[2]
		provider := "github"
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

type githubIssueItem struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	PullRequest json.RawMessage `json:"pull_request"`
}

func runExternalCommand(ctx context.Context, name string, args []string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	trace := ""
	if debuglog.Enabled() {
		trace = debuglog.NewTrace("exec")
		debuglog.LogCommand(trace, debuglog.FormatCommand(name, args))
	}
	err := cmd.Run()
	if debuglog.Enabled() {
		debuglog.LogStdoutLines(trace, stdout.String())
		debuglog.LogStderrLines(trace, stderr.String())
		debuglog.LogExit(trace, debuglog.ExitCode(err))
	}
	return stdout.String(), stderr.String(), err
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
	stdout, stderr, err := runExternalCommand(ctx, "gh", args)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg != "" {
			return nil, fmt.Errorf("gh api failed: %s", msg)
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}
	return parseGitHubIssues([]byte(stdout))
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
	stdout, stderr, err := runExternalCommand(ctx, "gh", args)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg != "" {
			return issueSummary{}, fmt.Errorf("gh api failed: %s", msg)
		}
		return issueSummary{}, fmt.Errorf("gh api failed: %w", err)
	}
	var item githubIssueItem
	if err := json.Unmarshal([]byte(stdout), &item); err != nil {
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

func issueTitleFromLabel(label string, number int) string {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return ""
	}
	prefix := fmt.Sprintf("#%d", number)
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	title := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	return title
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
			Value: encodeReviewSelection(pr),
		})
	}
	return choices
}

type reviewRepoChoice struct {
	Label    string
	Value    string
	Provider string
	Host     string
	Owner    string
	Repo     string
	RepoURL  string
}

type prSummary struct {
	Number   int
	Title    string
	HeadRef  string
	HeadRepo string
	BaseRepo string
}

func runCreateReview(ctx context.Context, rootDir, prURL string, noPrompt bool, prefetch *prefetcher.Prefetcher) error {
	prefetch = prefetcher.Ensure(prefetch, defaultPrefetchTimeout)
	prURL = strings.TrimSpace(prURL)
	if prURL == "" {
		if noPrompt {
			return fmt.Errorf("PR URL is required when --no-prompt is set")
		}
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("interactive review picker requires a TTY")
		}
		return runCreateReviewPicker(ctx, rootDir, noPrompt, prefetch)
	}

	req, err := parsePRURL(prURL)
	if err != nil {
		return err
	}

	provider, err := providerByName(req.Provider)
	if err != nil {
		return err
	}
	pr, err := provider.FetchPR(ctx, req.Host, req.Owner, req.Repo, req.Number)
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
	prefetchStarted := false
	if _, exists, err := repo.Exists(rootDir, repoURL); err != nil {
		return err
	} else if exists {
		started, err := prefetch.Start(ctx, rootDir, repoURL)
		if err != nil {
			return err
		}
		prefetchStarted = started
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Inputs")
	renderer.Bullet(fmt.Sprintf("provider: %s (%s)", strings.ToLower(req.Provider), req.Host))
	renderer.Bullet(fmt.Sprintf("repo: %s/%s", baseOwner, baseRepo))
	if prefetchStarted {
		renderTreeLines(renderer, []string{"prefetch: git fetch origin (background)"}, treeLineNormal)
	}
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
		if _, err := prefetch.Start(ctx, rootDir, repoURL); err != nil {
			return err
		}
	}

	workspaceID := formatReviewWorkspaceID(baseOwner, baseRepo, pr.Number)
	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := create.CreateWorkspace(ctx, rootDir, workspaceID, workspace.Metadata{
		Description: description,
		Mode:        workspace.MetadataModeReview,
		SourceURL:   prURL,
	})
	if err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return create.FailWorkspaceMetadata(err, rollbackErr)
		}
		return err
	}

	store, err := repo.Open(ctx, rootDir, repoURL, false)
	if err != nil {
		return err
	}
	if err := prefetch.Wait(ctx, repoURL); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("prefetch failed: %w (rollback failed: %v)", err, rollbackErr)
		}
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
	if err := rebuildManifest(ctx, rootDir); err != nil {
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestions(renderer, useColor, []string{
		"gwst open",
	})
	return nil
}

func runCreateReviewPicker(ctx context.Context, rootDir string, noPrompt bool, prefetch *prefetcher.Prefetcher) error {
	prefetch = prefetcher.Ensure(prefetch, defaultPrefetchTimeout)
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	repoChoices, err := buildReviewRepoChoices(rootDir)
	if err != nil {
		return err
	}
	if len(repoChoices) == 0 {
		return fmt.Errorf("no repos with supported review providers found")
	}

	promptChoices, repoByValue := toPromptChoices(repoChoices)
	loadReview := func(value string) ([]ui.PromptChoice, error) {
		selected, ok := repoByValue[value]
		if !ok {
			return nil, fmt.Errorf("selected repo not found")
		}
		provider, err := providerByName(selected.Provider)
		if err != nil {
			return nil, err
		}
		prs, err := provider.FetchPRs(ctx, selected.Host, selected.Owner, selected.Repo)
		if err != nil {
			return nil, err
		}
		return buildPRChoices(prs), nil
	}
	onReposResolved := func(repos []string) {
		for _, repoSpec := range repos {
			_, _ = prefetch.Start(ctx, rootDir, repoSpec)
		}
	}
	mode, _, _, _, _, reviewRepo, reviewPRs, _, _, _, err := ui.PromptCreateFlow("gwst create", "review", "", "", nil, nil, nil, nil, promptChoices, nil, loadReview, nil, nil, onReposResolved, nil, theme, useColor, "")
	if err != nil {
		return err
	}
	if mode != "review" {
		return fmt.Errorf("unknown mode: %s", mode)
	}
	return runCreateReviewSelected(ctx, rootDir, noPrompt, reviewRepo, reviewPRs, prefetch)
}

func runCreateReviewSelected(ctx context.Context, rootDir string, noPrompt bool, repoSpec string, selectedPRs []string, prefetch *prefetcher.Prefetcher) error {
	prefetch = prefetcher.Ensure(prefetch, defaultPrefetchTimeout)
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
	if _, err := prefetch.Start(ctx, rootDir, selectedRepo.RepoURL); err != nil {
		return err
	}
	if err := prefetch.Wait(ctx, selectedRepo.RepoURL); err != nil {
		return err
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
		pr, err := decodeReviewSelection(value)
		if err != nil {
			failure = err
			failureID = value
			break
		}
		if !strings.EqualFold(strings.TrimSpace(pr.HeadRepo), strings.TrimSpace(pr.BaseRepo)) {
			failure = fmt.Errorf("fork PRs are not supported: %s", pr.HeadRepo)
			failureID = fmt.Sprintf("PR-%d", pr.Number)
			break
		}
		description := pr.Title
		workspaceID := formatReviewWorkspaceID(selectedRepo.Owner, selectedRepo.Repo, pr.Number)
		output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
		sourceURL := buildPRURLFromParts(selectedRepo.Host, selectedRepo.Owner, selectedRepo.Repo, pr.Number)
		wsDir, err := create.CreateWorkspace(ctx, rootDir, workspaceID, workspace.Metadata{
			Description: description,
			Mode:        workspace.MetadataModeReview,
			SourceURL:   sourceURL,
		})
		if err != nil {
			if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
				failure = create.FailWorkspaceMetadata(err, rollbackErr)
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
		trackingExists, err := remoteTrackingExists(ctx, store.StorePath, pr.HeadRef)
		if err != nil {
			failure = err
			failureID = workspaceID
			break
		}
		if !trackingExists {
			if err := fetchPRHead(ctx, store.StorePath, pr.HeadRef); err != nil {
				failure = err
				failureID = workspaceID
				break
			}
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
		if err := rebuildManifest(ctx, rootDir); err != nil {
			return err
		}
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
			Label:    label,
			Value:    value,
			Provider: "github",
			Host:     host,
			Owner:    owner,
			Repo:     repoName,
			RepoURL:  repoURL,
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
	stdout, stderr, err := runExternalCommand(ctx, "gh", args)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg != "" {
			return prSummary{}, fmt.Errorf("gh api failed: %s", msg)
		}
		return prSummary{}, fmt.Errorf("gh api failed: %w", err)
	}
	var item githubPRItem
	if err := json.Unmarshal([]byte(stdout), &item); err != nil {
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
	stdout, stderr, err := runExternalCommand(ctx, "gh", args)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg != "" {
			return nil, fmt.Errorf("gh api failed: %s", msg)
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}
	return parseGitHubPRs([]byte(stdout))
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

func remoteTrackingExists(ctx context.Context, storePath, branch string) (bool, error) {
	if strings.TrimSpace(storePath) == "" {
		return false, fmt.Errorf("store path is required")
	}
	if strings.TrimSpace(branch) == "" {
		return false, fmt.Errorf("branch is required")
	}
	remoteRef := fmt.Sprintf("refs/remotes/origin/%s", branch)
	_, exists, err := gitcmd.ShowRef(ctx, storePath, remoteRef)
	if err != nil {
		return false, err
	}
	return exists, nil
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
	return workspace.AddWithBranch(ctx, rootDir, workspaceID, repoURL, "", branch, baseRef, false)
}

func addIssueWorktreeWithRemoteInfo(ctx context.Context, rootDir, workspaceID, repoURL, branch, baseRef, storePath string, remoteExists bool) (workspace.Repo, error) {
	if strings.TrimSpace(storePath) == "" {
		return workspace.Repo{}, fmt.Errorf("store path is required")
	}
	localExists, err := localBranchExists(ctx, storePath, branch)
	if err != nil {
		return workspace.Repo{}, err
	}
	if !localExists && remoteExists {
		remoteRef := fmt.Sprintf("refs/remotes/origin/%s", branch)
		trackingExists, err := remoteTrackingExists(ctx, storePath, branch)
		if err != nil {
			return workspace.Repo{}, err
		}
		if !trackingExists {
			if err := fetchRemoteBranch(ctx, storePath, branch); err != nil {
				return workspace.Repo{}, err
			}
		}
		return workspace.AddWithTrackingBranch(ctx, rootDir, workspaceID, repoURL, "", branch, remoteRef, false)
	}
	return workspace.AddWithBranch(ctx, rootDir, workspaceID, repoURL, "", branch, baseRef, false)
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

func encodeReviewSelection(pr prSummary) string {
	escape := url.QueryEscape
	return strings.Join([]string{
		strconv.Itoa(pr.Number),
		escape(pr.HeadRef),
		escape(pr.HeadRepo),
		escape(pr.BaseRepo),
		escape(pr.Title),
	}, "|")
}

func decodeReviewSelection(value string) (prSummary, error) {
	parts := strings.Split(value, "|")
	if len(parts) == 1 {
		num, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return prSummary{}, fmt.Errorf("invalid PR selection: %s", value)
		}
		return prSummary{}, fmt.Errorf("missing PR metadata for #%d; re-run selection", num)
	}
	if len(parts) != 5 {
		return prSummary{}, fmt.Errorf("invalid PR selection: %s", value)
	}
	num, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return prSummary{}, fmt.Errorf("invalid PR number: %s", parts[0])
	}
	unescape := func(v string) string {
		out, err := url.QueryUnescape(v)
		if err != nil {
			return v
		}
		return out
	}
	return prSummary{
		Number:   num,
		HeadRef:  strings.TrimSpace(unescape(parts[1])),
		HeadRepo: strings.TrimSpace(unescape(parts[2])),
		BaseRepo: strings.TrimSpace(unescape(parts[3])),
		Title:    strings.TrimSpace(unescape(parts[4])),
	}, nil
}

func runReview(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || (len(args) == 1 && isHelpArg(args[0])) {
		printCreateHelp(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gwst create --review [<PR URL>]")
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
	provider, err := providerByName(req.Provider)
	if err != nil {
		return err
	}
	pr, err := provider.FetchPR(ctx, req.Host, req.Owner, req.Repo, req.Number)
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
	wsDir, err := create.CreateWorkspace(ctx, rootDir, workspaceID, workspace.Metadata{
		Description: description,
		Mode:        workspace.MetadataModeReview,
		SourceURL:   raw,
	})
	if err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return create.FailWorkspaceMetadata(err, rollbackErr)
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
	if err := rebuildManifest(ctx, rootDir); err != nil {
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestions(renderer, useColor, []string{
		"gwst open",
	})
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

func buildIssueURLFromParts(host, owner, repo string, number int) string {
	repoName := strings.TrimSuffix(repo, ".git")
	return fmt.Sprintf("https://%s/%s/%s/issues/%d", host, owner, repoName, number)
}

func buildPRURLFromParts(host, owner, repo string, number int) string {
	repoName := strings.TrimSuffix(repo, ".git")
	return fmt.Sprintf("https://%s/%s/%s/pull/%d", host, owner, repoName, number)
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
	destRef := fmt.Sprintf("refs/remotes/gwst-review/%s", workspaceID)
	spec := fmt.Sprintf("%s:%s", srcRef, destRef)
	gitcmd.Logf("git fetch origin %s", spec)
	if _, err := gitcmd.Run(ctx, []string{"fetch", "origin", spec}, gitcmd.Options{Dir: storePath}); err != nil {
		return "", err
	}
	return destRef, nil
}

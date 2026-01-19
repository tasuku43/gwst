package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/create"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/template"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/output"
	"github.com/tasuku43/gwst/internal/ui"
)

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
	prefetch := newPrefetcher(defaultPrefetchTimeout)

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
		onReposResolved := func(repos []string) {
			for _, repoSpec := range repos {
				_, _ = prefetch.start(ctx, rootDir, repoSpec)
			}
		}
		mode, tmplName, tmplWorkspaceID, tmplDesc, tmplBranches, reviewRepo, reviewPRs, issueRepo, issueSelections, repoSelected, err := ui.PromptCreateFlow("gwst create", "", "", "", templateNames, tmplErr, repoChoices, repoErr, reviewPrompt, issuePrompt, loadReview, loadIssue, loadTemplateRepos, onReposResolved, validateBranch, theme, useColor, "")
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
			return runCreateTemplateWithInputs(ctx, rootDir, inputs, noPrompt, prefetch)
		case "review":
			if err := runCreateReviewSelected(ctx, rootDir, noPrompt, reviewRepo, reviewPRs, prefetch); err != nil {
				return err
			}
			return nil
		case "issue":
			if err := runCreateIssueSelected(ctx, rootDir, noPrompt, issueRepo, issueSelections, prefetch); err != nil {
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
			return runCreateRepoWithInputs(ctx, rootDir, inputs, noPrompt, prefetch)
		default:
			return fmt.Errorf("unknown mode: %s", mode)
		}
	}

	remaining := createFlags.Args()
	if templateMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gwst create --template <name> [<WORKSPACE_ID>]")
		}
		if len(remaining) == 1 {
			if workspaceID != "" && workspaceID != remaining[0] {
				return fmt.Errorf("workspace id is specified twice: %s and %s", workspaceID, remaining[0])
			}
			workspaceID = remaining[0]
		}
		return runCreateTemplate(ctx, rootDir, templateName.value, workspaceID, noPrompt, prefetch)
	}
	if reviewMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gwst create --review [<PR URL>]")
		}
		if workspaceID != "" || branch != "" || baseRef != "" {
			return fmt.Errorf("--workspace-id, --branch, and --base are not valid with --review")
		}
		prURL := ""
		if len(remaining) == 1 {
			prURL = remaining[0]
		}
		return runCreateReview(ctx, rootDir, prURL, noPrompt, prefetch)
	}
	if issueMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gwst create --issue [<ISSUE_URL>] [--workspace-id <id>] [--branch <name>] [--base <ref>]")
		}
		issueURL := ""
		if len(remaining) == 1 {
			issueURL = remaining[0]
		}
		return runCreateIssue(ctx, rootDir, issueURL, workspaceID, branch, baseRef, noPrompt, prefetch)
	}
	if repoMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gwst create --repo [<repo>]")
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
			onReposResolved := func(repos []string) {
				for _, repoSpec := range repos {
					_, _ = prefetch.start(ctx, rootDir, repoSpec)
				}
			}
			mode, _, tmplWorkspaceID, tmplDesc, tmplBranches, _, _, _, _, repoSelected, err := ui.PromptCreateFlow("gwst create", "repo", workspaceID, "", nil, nil, repoChoices, repoErr, nil, nil, nil, nil, nil, onReposResolved, func(v string) error {
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
			return runCreateRepoWithInputs(ctx, rootDir, inputs, noPrompt, prefetch)
		}
		if !noPrompt {
			if !isatty.IsTerminal(os.Stdin.Fd()) {
				return fmt.Errorf("repo prompts require a TTY")
			}
			theme := ui.DefaultTheme()
			useColor := isatty.IsTerminal(os.Stdout.Fd())
			onReposResolved := func(repos []string) {
				for _, repoSpec := range repos {
					_, _ = prefetch.start(ctx, rootDir, repoSpec)
				}
			}
			mode, _, tmplWorkspaceID, tmplDesc, tmplBranches, _, _, _, _, repoSelected, err := ui.PromptCreateFlow("gwst create", "repo", workspaceID, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, onReposResolved, func(v string) error {
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
			return runCreateRepoWithInputs(ctx, rootDir, inputs, noPrompt, prefetch)
		}
		inputs := createRepoInputs{
			repos:       []string{repoSpec},
			workspaceID: workspaceID,
			fromFlow:    false,
		}
		return runCreateRepoWithInputs(ctx, rootDir, inputs, noPrompt, prefetch)
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
		return fmt.Errorf("usage: gwst create --template <name> [<WORKSPACE_ID>]")
	}

	workspaceID := ""
	if newFlags.NArg() == 1 {
		workspaceID = newFlags.Arg(0)
	}

	prefetch := newPrefetcher(defaultPrefetchTimeout)
	return runCreateTemplate(ctx, rootDir, templateName, workspaceID, noPrompt, prefetch)
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

func runCreateTemplate(ctx context.Context, rootDir, templateName, workspaceID string, noPrompt bool, prefetch *prefetcher) error {
	inputs := createTemplateInputs{
		templateName: templateName,
		workspaceID:  workspaceID,
	}
	return runCreateTemplateWithInputs(ctx, rootDir, inputs, noPrompt, prefetch)
}

func runCreateTemplateWithInputs(ctx context.Context, rootDir string, inputs createTemplateInputs, noPrompt bool, prefetch *prefetcher) error {
	prefetch = ensurePrefetcher(prefetch)
	templateName := strings.TrimSpace(inputs.templateName)
	workspaceID := strings.TrimSpace(inputs.workspaceID)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	description := strings.TrimSpace(inputs.description)
	branches := inputs.branches
	fromFlow := inputs.fromFlow

	if !noPrompt && !fromFlow {
		templateNames, tmplErr := loadTemplateNames(rootDir)
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
		onReposResolved := func(repos []string) {
			for _, repoSpec := range repos {
				_, _ = prefetch.start(ctx, rootDir, repoSpec)
			}
		}
		mode, tmplName, tmplWorkspaceID, tmplDesc, tmplBranches, _, _, _, _, _, err := ui.PromptCreateFlow("gwst create", "template", workspaceID, templateName, templateNames, tmplErr, nil, nil, nil, nil, nil, nil, loadTemplateRepos, onReposResolved, validateBranch, theme, useColor, "")
		if err != nil {
			return err
		}
		if mode != "template" {
			return fmt.Errorf("unknown mode: %s", mode)
		}
		templateName = tmplName
		workspaceID = tmplWorkspaceID
		description = tmplDesc
		branches = tmplBranches
		fromFlow = true
	}

	if templateName == "" || workspaceID == "" {
		if noPrompt {
			return fmt.Errorf("template name and workspace id are required without prompt")
		}
		return fmt.Errorf("template name and workspace id are required")
	}
	if !noPrompt && !fromFlow {
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
	if _, err := prefetch.startAll(ctx, rootDir, tmpl.Repos); err != nil {
		return err
	}
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	if !noPrompt && !fromFlow {
		branches, err = promptTemplateBranches(ctx, tmpl, workspaceID, theme, useColor)
		if err != nil {
			return err
		}
	}

	startSteps(renderer)
	if err := ensureRepoGet(ctx, rootDir, missing, noPrompt, theme, useColor); err != nil {
		return err
	}
	if _, err := prefetch.startAll(ctx, rootDir, missing); err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := create.CreateWorkspace(ctx, rootDir, workspaceID, workspace.Metadata{
		Description:  description,
		Mode:         workspace.MetadataModeTemplate,
		TemplateName: templateName,
	})
	if err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return create.FailWorkspaceMetadata(err, rollbackErr)
		}
		return err
	}

	if err := prefetch.waitAll(ctx, tmpl.Repos); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("prefetch failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}
	if err := create.ApplyTemplate(ctx, rootDir, workspaceID, tmpl, branches, func(repoSpec string, index, total int) {
		output.Step(formatStepWithIndex("worktree add", displayRepoName(repoSpec), worktreeDest(rootDir, workspaceID, repoSpec), index+1, total))
	}); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("apply template failed: %w (rollback failed: %v)", err, rollbackErr)
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

func runCreateRepoWithInputs(ctx context.Context, rootDir string, inputs createRepoInputs, noPrompt bool, prefetch *prefetcher) error {
	prefetch = ensurePrefetcher(prefetch)
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
			return fmt.Errorf("no repos found; run gwst repo get first")
		}
		selected, err := ui.PromptChoiceSelect("gwst create", "repo", choices, theme, useColor)
		if err != nil {
			return err
		}
		repoSpecs = template.NormalizeRepos([]string{selected})
	}
	if len(repoSpecs) != 1 {
		return fmt.Errorf("exactly one repo is required")
	}
	if _, err := prefetch.startAll(ctx, rootDir, repoSpecs); err != nil {
		return err
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
	if _, err := prefetch.startAll(ctx, rootDir, missing); err != nil {
		return err
	}

	output.Step(formatStep("create workspace", workspaceID, relPath(rootDir, workspace.WorkspaceDir(rootDir, workspaceID))))
	wsDir, err := create.CreateWorkspace(ctx, rootDir, workspaceID, workspace.Metadata{
		Description: description,
		Mode:        workspace.MetadataModeRepo,
	})
	if err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return create.FailWorkspaceMetadata(err, rollbackErr)
		}
		return err
	}

	if err := prefetch.waitAll(ctx, tmpl.Repos); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("prefetch failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}
	if err := create.ApplyTemplate(ctx, rootDir, workspaceID, tmpl, branches, func(repoSpec string, index, total int) {
		output.Step(formatStepWithIndex("worktree add", displayRepoName(repoSpec), worktreeDest(rootDir, workspaceID, repoSpec), index+1, total))
	}); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("apply repo selection failed: %w (rollback failed: %v)", err, rollbackErr)
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
		{Label: "issue", Value: "issue", Description: "From an issue (multi-select, GitHub only)"},
		{Label: "review", Value: "review", Description: "From a review request (multi-select, GitHub only)"},
		{Label: "template", Value: "template", Description: "From template"},
	}
	return ui.PromptChoiceSelect("gwst create", "mode", choices, theme, useColor)
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

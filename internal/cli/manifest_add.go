package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gion/internal/app/manifestplan"
	"github.com/tasuku43/gion/internal/domain/manifest"
	"github.com/tasuku43/gion/internal/domain/preset"
	"github.com/tasuku43/gion/internal/domain/repo"
	"github.com/tasuku43/gion/internal/domain/workspace"
	"github.com/tasuku43/gion/internal/infra/paths"
	"github.com/tasuku43/gion/internal/ui"
)

func runManifestAdd(ctx context.Context, rootDir string, args []string, globalNoPrompt bool) error {
	addFlags := flag.NewFlagSet("manifest add", flag.ContinueOnError)
	var presetName stringFlag
	var reviewFlag boolFlag
	var issueFlag boolFlag
	var repoFlag stringFlag
	var branch string
	var baseRef string
	var helpFlag bool
	var noApply bool
	var noPromptFlag bool
	var workspaceIDFlag stringFlag
	addFlags.Var(&presetName, "preset", "preset name")
	addFlags.Var(&reviewFlag, "review", "add review workspace from PR")
	addFlags.Var(&issueFlag, "issue", "add issue workspace from issue")
	addFlags.Var(&repoFlag, "repo", "add workspace from a repo")
	addFlags.Var(&workspaceIDFlag, "workspace-id", "not supported (use positional WORKSPACE_ID)")
	addFlags.StringVar(&branch, "branch", "", "branch name")
	addFlags.StringVar(&baseRef, "base", "", "base ref")
	addFlags.BoolVar(&noApply, "no-apply", false, "do not run gion apply")
	addFlags.BoolVar(&noPromptFlag, "no-prompt", false, "disable interactive prompt")
	addFlags.BoolVar(&helpFlag, "help", false, "show help")
	addFlags.BoolVar(&helpFlag, "h", false, "show help")
	addFlags.SetOutput(os.Stdout)
	addFlags.Usage = func() {
		printManifestAddHelp(os.Stdout)
	}
	if err := addFlags.Parse(normalizeManifestAddArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printManifestAddHelp(os.Stdout)
		return nil
	}
	if workspaceIDFlag.set {
		return fmt.Errorf("--workspace-id is not supported; use positional <WORKSPACE_ID>")
	}

	branch = strings.TrimSpace(branch)
	baseRef = strings.TrimSpace(baseRef)
	presetName.value = strings.TrimSpace(presetName.value)
	repoFlag.value = strings.TrimSpace(repoFlag.value)
	noPrompt := globalNoPrompt || noPromptFlag

	presetMode := presetName.set
	reviewMode := reviewFlag.value
	issueMode := issueFlag.value
	repoMode := repoFlag.set
	modeCount := 0
	if presetMode {
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
		return fmt.Errorf("specify exactly one mode: --preset, --review, --issue, or --repo")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	manifestPath := manifest.Path(rootDir)
	originalBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", manifest.FileName, err)
	}

	apply := func(updated manifest.File, showInputs func(*ui.Renderer), addedWorkspaceIDs []string) error {
		return applyManifestMutation(ctx, rootDir, updated, manifestMutationOptions{
			NoApply:       noApply,
			NoPrompt:      noPrompt,
			OriginalBytes: originalBytes,
			Hooks: manifestMutationHooks{
				ShowPrelude: showInputs,
				RenderNoApply: func(r *ui.Renderer) {
					r.Section("Result")
					r.Bullet(fmt.Sprintf("updated %s", manifest.FileName))
					r.Blank()
					r.Section("Suggestion")
					r.Bullet("gion apply")
				},
				RenderNoChanges: func(r *ui.Renderer) {
					r.Section("Result")
					r.Bullet(fmt.Sprintf("updated %s", manifest.FileName))
					r.Bullet("no changes")
				},
				RenderInfoBeforeApply: func(r *ui.Renderer, plan manifestplan.Result, planOK bool) {
					r.Section("Info")
					r.Bullet(fmt.Sprintf("manifest: updated %s", manifest.FileName))
					if planOK && planIncludesChangesOutsideWorkspaceIDs(plan, addedWorkspaceIDs) {
						r.BulletWarn("apply: plan includes unrelated changes")
					}
				},
			},
		})
	}

	// Interactive mode picker / unified prompt flow.
	if modeCount == 0 {
		if noPrompt {
			return fmt.Errorf("mode is required when --no-prompt is set")
		}
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("interactive mode picker requires a TTY")
		}

		presetNames, tmplErr := loadPresetNames(rootDir)
		repoChoices, repoErr := buildPresetRepoChoices(rootDir)
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
		loadPresetRepos := func(name string) ([]string, error) {
			file, err := preset.Load(rootDir)
			if err != nil {
				return nil, err
			}
			tmpl, ok := file.Presets[name]
			if !ok {
				return nil, fmt.Errorf("preset not found: %s", name)
			}
			return append([]string(nil), tmpl.Repos...), nil
		}
		validateBranch := func(v string) error {
			return workspace.ValidateBranchName(ctx, v)
		}
		validateWorkspaceID := func(v string) error {
			return workspace.ValidateWorkspaceID(ctx, v)
		}

		mode, tmplName, tmplWorkspaceID, tmplDesc, tmplBranches, reviewRepo, reviewPRs, issueRepo, issueSelections, repoSelected, err := ui.PromptCreateFlow(
			"gion manifest add",
			"",
			"",
			"",
			presetNames,
			tmplErr,
			repoChoices,
			repoErr,
			reviewPrompt,
			issuePrompt,
			loadReview,
			loadIssue,
			loadPresetRepos,
			nil,
			validateBranch,
			validateWorkspaceID,
			theme,
			useColor,
			"",
		)
		if err != nil {
			return err
		}

		switch mode {
		case "preset":
			return manifestAddPreset(ctx, rootDir, tmplName, tmplWorkspaceID, tmplDesc, tmplBranches, baseRef, apply, originalBytes)
		case "repo":
			return manifestAddRepo(ctx, rootDir, repoSelected, tmplWorkspaceID, tmplDesc, tmplBranches, baseRef, apply, originalBytes)
		case "review":
			return manifestAddReviewSelected(ctx, rootDir, reviewRepo, reviewPRs, apply, originalBytes)
		case "issue":
			return manifestAddIssueSelected(ctx, rootDir, issueRepo, issueSelections, baseRef, apply, originalBytes)
		default:
			return fmt.Errorf("unknown mode: %s", mode)
		}
	}

	remaining := addFlags.Args()
	if presetMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gion manifest add --preset <name> [<WORKSPACE_ID>]")
		}
		workspaceID := ""
		if len(remaining) == 1 {
			workspaceID = remaining[0]
		}
		if strings.TrimSpace(workspaceID) == "" && noPrompt {
			return fmt.Errorf("workspace id is required when --no-prompt is set")
		}
		if strings.TrimSpace(workspaceID) == "" {
			return fmt.Errorf("workspace id is required")
		}
		if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
			return err
		}
		file, err := preset.Load(rootDir)
		if err != nil {
			return err
		}
		tmpl, ok := file.Presets[presetName.value]
		if !ok {
			return fmt.Errorf("preset not found: %s", presetName.value)
		}
		branches := make([]string, len(tmpl.Repos))
		for i := range branches {
			branches[i] = workspaceID
		}

		renderInputs := func(r *ui.Renderer) {
			r.Section("Inputs")
			r.Bullet("mode: preset")
			r.Bullet(fmt.Sprintf("preset: %s", presetName.value))
			r.Bullet(fmt.Sprintf("workspace id: %s", workspaceID))
			r.Bullet("branches")
			var branchLines []string
			for i, repoSpec := range tmpl.Repos {
				branchLines = append(branchLines, fmt.Sprintf("%s: %s", displayRepoName(repoSpec), branches[i]))
			}
			renderTreeLines(r, branchLines, treeLineNormal)
		}
		return manifestAddPresetWithFile(ctx, rootDir, presetName.value, workspaceID, "", tmpl, branches, baseRef, apply, renderInputs)
	}

	if repoMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gion manifest add --repo [<repo>] [<WORKSPACE_ID>]")
		}
		repoSpec := strings.TrimSpace(repoFlag.value)
		workspaceID := ""
		if len(remaining) == 1 {
			workspaceID = strings.TrimSpace(remaining[0])
		}
		if strings.TrimSpace(repoSpec) == "" && noPrompt {
			return fmt.Errorf("--repo requires a repo argument when --no-prompt is set")
		}
		if strings.TrimSpace(workspaceID) == "" && noPrompt {
			return fmt.Errorf("workspace id is required when --no-prompt is set")
		}
		if strings.TrimSpace(repoSpec) == "" || strings.TrimSpace(workspaceID) == "" {
			return fmt.Errorf("repo and workspace id are required")
		}
		if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
			return err
		}
		repoSpecNorm, err := normalizeRepoSpec(repoSpec)
		if err != nil {
			return err
		}
		branchValue := strings.TrimSpace(branch)
		if branchValue == "" {
			branchValue = workspaceID
		}
		if err := workspace.ValidateBranchName(ctx, branchValue); err != nil {
			return err
		}
		renderInputs := func(r *ui.Renderer) {
			r.Section("Inputs")
			r.Bullet("mode: repo")
			r.Bullet(fmt.Sprintf("repo: %s", displayRepoSpec(repoSpecNorm)))
			r.Bullet(fmt.Sprintf("workspace id: %s", workspaceID))
			r.Bullet(fmt.Sprintf("branch: %s", branchValue))
			if baseRef != "" {
				r.Bullet(fmt.Sprintf("base: %s", baseRef))
			}
		}
		return manifestAddRepoWithSpec(ctx, rootDir, repoSpecNorm, workspaceID, "", branchValue, baseRef, apply, renderInputs)
	}

	if reviewMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gion manifest add --review [<PR URL>]")
		}
		if branch != "" || baseRef != "" {
			return fmt.Errorf("--branch and --base are not valid with --review")
		}
		prURL := ""
		if len(remaining) == 1 {
			prURL = remaining[0]
		}
		if strings.TrimSpace(prURL) == "" {
			if noPrompt {
				return fmt.Errorf("PR URL is required when --no-prompt is set")
			}
			return fmt.Errorf("PR URL is required")
		}
		return manifestAddReviewURL(ctx, rootDir, prURL, apply)
	}

	if issueMode {
		if len(remaining) > 1 {
			return fmt.Errorf("usage: gion manifest add --issue <ISSUE_URL> [--branch <name>] [--base <ref>]")
		}
		issueURL := ""
		if len(remaining) == 1 {
			issueURL = remaining[0]
		}
		if strings.TrimSpace(issueURL) == "" {
			if noPrompt {
				return fmt.Errorf("issue URL is required when --no-prompt is set")
			}
			return fmt.Errorf("issue URL is required")
		}
		return manifestAddIssueURL(ctx, rootDir, issueURL, branch, baseRef, noPrompt, apply)
	}

	return fmt.Errorf("mode is required")
}

func normalizeRepoSpec(raw string) (string, error) {
	_, trimmed, err := repo.Normalize(raw)
	if err != nil {
		return "", err
	}
	return trimmed, nil
}

func manifestAddPreset(ctx context.Context, rootDir, presetName, workspaceID, description string, branches []string, baseRef string, apply func(manifest.File, func(*ui.Renderer), []string) error, _ []byte) error {
	file, err := preset.Load(rootDir)
	if err != nil {
		return err
	}
	tmpl, ok := file.Presets[presetName]
	if !ok {
		return fmt.Errorf("preset not found: %s", presetName)
	}
	return manifestAddPresetWithFile(ctx, rootDir, presetName, workspaceID, description, tmpl, branches, baseRef, apply, nil)
}

func manifestAddPresetWithFile(ctx context.Context, rootDir, presetName, workspaceID, description string, tmpl preset.Preset, branches []string, baseRef string, apply func(manifest.File, func(*ui.Renderer), []string) error, showInputs func(*ui.Renderer)) error {
	if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
		return err
	}
	desired, err := manifest.Load(rootDir)
	if err != nil {
		return err
	}

	if _, exists := desired.Workspaces[workspaceID]; exists {
		return fmt.Errorf("workspace already exists in %s: %s", manifest.FileName, workspaceID)
	}
	wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
	if exists, err := paths.DirExists(wsDir); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("workspace exists on filesystem but missing in %s: %s (suggest: gion import)", manifest.FileName, workspaceID)
	}

	var repos []manifest.Repo
	for i, repoSpec := range tmpl.Repos {
		spec, _, err := repo.Normalize(repoSpec)
		if err != nil {
			return err
		}
		branchValue := workspaceID
		if len(branches) == len(tmpl.Repos) && i < len(branches) && strings.TrimSpace(branches[i]) != "" {
			branchValue = strings.TrimSpace(branches[i])
		}
		if err := workspace.ValidateBranchName(ctx, branchValue); err != nil {
			return err
		}
		repos = append(repos, manifest.Repo{
			Alias:   strings.TrimSpace(spec.Repo),
			RepoKey: strings.TrimSpace(spec.RepoKey),
			Branch:  branchValue,
			BaseRef: strings.TrimSpace(baseRef),
		})
	}

	desired.Workspaces[workspaceID] = manifest.Workspace{
		Description: strings.TrimSpace(description),
		Mode:        workspace.MetadataModePreset,
		PresetName:  strings.TrimSpace(presetName),
		Repos:       repos,
	}
	return apply(desired, showInputs, []string{workspaceID})
}

func manifestAddRepo(ctx context.Context, rootDir, repoSpec, workspaceID, description string, branches []string, baseRef string, apply func(manifest.File, func(*ui.Renderer), []string) error, _ []byte) error {
	repoSpecNorm, err := normalizeRepoSpec(repoSpec)
	if err != nil {
		return err
	}
	branchValue := workspaceID
	if len(branches) == 1 && strings.TrimSpace(branches[0]) != "" {
		branchValue = strings.TrimSpace(branches[0])
	}
	if err := workspace.ValidateBranchName(ctx, branchValue); err != nil {
		return err
	}
	return manifestAddRepoWithSpec(ctx, rootDir, repoSpecNorm, workspaceID, description, branchValue, baseRef, apply, nil)
}

func manifestAddRepoWithSpec(ctx context.Context, rootDir, repoSpec, workspaceID, description, branch, baseRef string, apply func(manifest.File, func(*ui.Renderer), []string) error, showInputs func(*ui.Renderer)) error {
	if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
		return err
	}
	desired, err := manifest.Load(rootDir)
	if err != nil {
		return err
	}
	if _, exists := desired.Workspaces[workspaceID]; exists {
		return fmt.Errorf("workspace already exists in %s: %s", manifest.FileName, workspaceID)
	}
	wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
	if exists, err := paths.DirExists(wsDir); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("workspace exists on filesystem but missing in %s: %s (suggest: gion import)", manifest.FileName, workspaceID)
	}

	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		return err
	}
	if err := workspace.ValidateBranchName(ctx, branch); err != nil {
		return err
	}

	desired.Workspaces[workspaceID] = manifest.Workspace{
		Description: strings.TrimSpace(description),
		Mode:        workspace.MetadataModeRepo,
		Repos: []manifest.Repo{
			{
				Alias:   strings.TrimSpace(spec.Repo),
				RepoKey: strings.TrimSpace(spec.RepoKey),
				Branch:  strings.TrimSpace(branch),
				BaseRef: strings.TrimSpace(baseRef),
			},
		},
	}
	return apply(desired, showInputs, []string{workspaceID})
}

func manifestAddReviewURL(ctx context.Context, rootDir, prURL string, apply func(manifest.File, func(*ui.Renderer), []string) error) error {
	prURL = strings.TrimSpace(prURL)
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
	baseOwner, baseRepo, ok := splitRepoFullName(pr.BaseRepo)
	if !ok {
		return fmt.Errorf("invalid base repo: %s", pr.BaseRepo)
	}

	workspaceID := formatReviewWorkspaceID(baseOwner, baseRepo, pr.Number)
	if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
		return err
	}
	description := pr.Title
	renderInputs := func(r *ui.Renderer) {
		r.Section("Inputs")
		r.Bullet("mode: review")
		r.Bullet(fmt.Sprintf("repo: %s/%s", baseOwner, baseRepo))
		r.Bullet(fmt.Sprintf("pull request: #%d", pr.Number))
		r.Bullet(fmt.Sprintf("workspace id: %s", workspaceID))
		r.Bullet(fmt.Sprintf("branch: %s", pr.HeadRef))
	}

	desired, err := manifest.Load(rootDir)
	if err != nil {
		return err
	}
	if _, exists := desired.Workspaces[workspaceID]; exists {
		return fmt.Errorf("workspace already exists in %s: %s", manifest.FileName, workspaceID)
	}
	wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
	if exists, err := paths.DirExists(wsDir); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("workspace exists on filesystem but missing in %s: %s (suggest: gion import)", manifest.FileName, workspaceID)
	}
	repoURL := buildRepoURLFromParts(req.Host, baseOwner, baseRepo)
	spec, _, err := repo.Normalize(repoURL)
	if err != nil {
		return err
	}
	if err := workspace.ValidateBranchName(ctx, pr.HeadRef); err != nil {
		return err
	}

	desired.Workspaces[workspaceID] = manifest.Workspace{
		Description: strings.TrimSpace(description),
		Mode:        workspace.MetadataModeReview,
		SourceURL:   prURL,
		Repos: []manifest.Repo{
			{
				Alias:   strings.TrimSpace(spec.Repo),
				RepoKey: strings.TrimSpace(spec.RepoKey),
				Branch:  strings.TrimSpace(pr.HeadRef),
				BaseRef: formatPRBaseRef(pr.BaseRef),
			},
		},
	}
	return apply(desired, renderInputs, []string{workspaceID})
}

func manifestAddReviewSelected(ctx context.Context, rootDir string, repoSpec string, selectedPRs []string, apply func(manifest.File, func(*ui.Renderer), []string) error, _ []byte) error {
	repoSpec = strings.TrimSpace(repoSpec)
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		return err
	}
	host := strings.TrimSpace(spec.Host)
	if host == "" {
		return fmt.Errorf("host is required")
	}

	desired, err := manifest.Load(rootDir)
	if err != nil {
		return err
	}
	updated := desired
	var warnings []string
	var addedWorkspaceIDs []string
	added := 0

	for _, raw := range selectedPRs {
		pr, err := decodeReviewSelection(raw)
		if err != nil {
			return err
		}
		if !strings.EqualFold(strings.TrimSpace(pr.HeadRepo), strings.TrimSpace(pr.BaseRepo)) {
			warnings = append(warnings, fmt.Sprintf("skipped PR #%d: fork PRs are not supported", pr.Number))
			continue
		}
		baseOwner, baseRepo, ok := splitRepoFullName(pr.BaseRepo)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("skipped PR #%d: invalid base repo: %s", pr.Number, pr.BaseRepo))
			continue
		}

		workspaceID := formatReviewWorkspaceID(baseOwner, baseRepo, pr.Number)
		if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped PR #%d: invalid workspace id: %s", pr.Number, err.Error()))
			continue
		}
		if _, exists := updated.Workspaces[workspaceID]; exists {
			warnings = append(warnings, fmt.Sprintf("skipped: workspace already exists in %s: %s", manifest.FileName, workspaceID))
			continue
		}
		wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
		if exists, err := paths.DirExists(wsDir); err != nil {
			return err
		} else if exists {
			warnings = append(warnings, fmt.Sprintf("skipped: workspace exists on filesystem but missing in %s: %s (suggest: gion import)", manifest.FileName, workspaceID))
			continue
		}
		if err := workspace.ValidateBranchName(ctx, pr.HeadRef); err != nil {
			return err
		}
		repoURL := buildRepoURLFromParts(host, baseOwner, baseRepo)
		repoSpecNorm, err := normalizeRepoSpec(repoURL)
		if err != nil {
			return err
		}
		repoNorm, _, err := repo.Normalize(repoSpecNorm)
		if err != nil {
			return err
		}
		updated.Workspaces[workspaceID] = manifest.Workspace{
			Description: strings.TrimSpace(pr.Title),
			Mode:        workspace.MetadataModeReview,
			SourceURL:   buildPRURLFromParts(host, baseOwner, baseRepo, pr.Number),
			Repos: []manifest.Repo{
				{
					Alias:   strings.TrimSpace(repoNorm.Repo),
					RepoKey: strings.TrimSpace(repoNorm.RepoKey),
					Branch:  strings.TrimSpace(pr.HeadRef),
					BaseRef: formatPRBaseRef(pr.BaseRef),
				},
			},
		}
		added++
		addedWorkspaceIDs = append(addedWorkspaceIDs, workspaceID)
	}

	if added == 0 {
		if len(warnings) > 0 {
			return fmt.Errorf("%s", warnings[0])
		}
		return fmt.Errorf("no selections")
	}

	showInputs := func(r *ui.Renderer) {
		if len(warnings) == 0 {
			return
		}
		renderWarningsSection(r, "warnings", warnings, false)
		r.Blank()
	}

	_ = spec
	return apply(updated, showInputs, addedWorkspaceIDs)
}

func manifestAddIssueURL(ctx context.Context, rootDir, issueURL, branch, baseRef string, noPrompt bool, apply func(manifest.File, func(*ui.Renderer), []string) error) error {
	issueURL = strings.TrimSpace(issueURL)
	req, err := parseIssueURL(issueURL)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(req.Provider), "github") {
		return fmt.Errorf("unsupported issue provider: %s", req.Provider)
	}
	provider, err := providerByName(req.Provider)
	if err != nil {
		return err
	}
	issue, err := provider.FetchIssue(ctx, req.Host, req.Owner, req.Repo, req.Number)
	if err != nil {
		return err
	}

	workspaceID := formatIssueWorkspaceID(req.Owner, req.Repo, req.Number)
	if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
		return err
	}
	branchValue := strings.TrimSpace(branch)
	if branchValue == "" {
		branchValue = fmt.Sprintf("issue/%d", req.Number)
	}
	if err := workspace.ValidateBranchName(ctx, branchValue); err != nil {
		return err
	}

	repoURL := buildRepoURLFromParts(req.Host, req.Owner, req.Repo)
	spec, _, err := repo.Normalize(repoURL)
	if err != nil {
		return err
	}

	renderInputs := func(r *ui.Renderer) {
		r.Section("Inputs")
		r.Bullet("mode: issue")
		r.Bullet(fmt.Sprintf("repo: %s/%s", req.Owner, req.Repo))
		r.Bullet(fmt.Sprintf("issue: #%d", req.Number))
		r.Bullet(fmt.Sprintf("workspace id: %s", workspaceID))
		r.Bullet(fmt.Sprintf("branch: %s", branchValue))
		if strings.TrimSpace(baseRef) != "" {
			r.Bullet(fmt.Sprintf("base: %s", strings.TrimSpace(baseRef)))
		}
	}

	desired, err := manifest.Load(rootDir)
	if err != nil {
		return err
	}
	if _, exists := desired.Workspaces[workspaceID]; exists {
		return fmt.Errorf("workspace already exists in %s: %s", manifest.FileName, workspaceID)
	}
	wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
	if exists, err := paths.DirExists(wsDir); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("workspace exists on filesystem but missing in %s: %s (suggest: gion import)", manifest.FileName, workspaceID)
	}

	desired.Workspaces[workspaceID] = manifest.Workspace{
		Description: strings.TrimSpace(issue.Title),
		Mode:        workspace.MetadataModeIssue,
		SourceURL:   issueURL,
		Repos: []manifest.Repo{
			{
				Alias:   strings.TrimSpace(spec.Repo),
				RepoKey: strings.TrimSpace(spec.RepoKey),
				Branch:  branchValue,
				BaseRef: strings.TrimSpace(baseRef),
			},
		},
	}

	_ = noPrompt
	return apply(desired, renderInputs, []string{workspaceID})
}

func manifestAddIssueSelected(ctx context.Context, rootDir string, repoSpec string, selections []ui.IssueSelection, baseRef string, apply func(manifest.File, func(*ui.Renderer), []string) error, _ []byte) error {
	repoSpec = strings.TrimSpace(repoSpec)
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		return err
	}
	if !isGitHubHost(spec.Host) {
		return fmt.Errorf("issue picker supports GitHub only for now: %s", spec.Host)
	}
	host := strings.TrimSpace(spec.Host)
	owner := strings.TrimSpace(spec.Owner)
	repoName := strings.TrimSpace(spec.Repo)

	desired, err := manifest.Load(rootDir)
	if err != nil {
		return err
	}
	updated := desired
	var warnings []string
	var addedWorkspaceIDs []string
	added := 0

	for _, sel := range selections {
		num, err := strconv.Atoi(strings.TrimSpace(sel.Value))
		if err != nil || num <= 0 {
			warnings = append(warnings, fmt.Sprintf("skipped issue: invalid number: %s", sel.Value))
			continue
		}
		workspaceID := formatIssueWorkspaceID(owner, repoName, num)
		if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped issue #%d: invalid workspace id: %s", num, err.Error()))
			continue
		}
		if _, exists := updated.Workspaces[workspaceID]; exists {
			warnings = append(warnings, fmt.Sprintf("skipped: workspace already exists in %s: %s", manifest.FileName, workspaceID))
			continue
		}
		wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
		if exists, err := paths.DirExists(wsDir); err != nil {
			return err
		} else if exists {
			warnings = append(warnings, fmt.Sprintf("skipped: workspace exists on filesystem but missing in %s: %s (suggest: gion import)", manifest.FileName, workspaceID))
			continue
		}

		branchValue := strings.TrimSpace(sel.Branch)
		if branchValue == "" {
			branchValue = fmt.Sprintf("issue/%d", num)
		}
		if err := workspace.ValidateBranchName(ctx, branchValue); err != nil {
			return err
		}

		title := issueTitleFromLabel(sel.Label, num)
		updated.Workspaces[workspaceID] = manifest.Workspace{
			Description: strings.TrimSpace(title),
			Mode:        workspace.MetadataModeIssue,
			SourceURL:   buildIssueURLFromParts(host, owner, repoName, num),
			Repos: []manifest.Repo{
				{
					Alias:   strings.TrimSpace(spec.Repo),
					RepoKey: strings.TrimSpace(spec.RepoKey),
					Branch:  branchValue,
					BaseRef: strings.TrimSpace(baseRef),
				},
			},
		}
		added++
		addedWorkspaceIDs = append(addedWorkspaceIDs, workspaceID)
	}

	if added == 0 {
		if len(warnings) > 0 {
			return fmt.Errorf("%s", warnings[0])
		}
		return fmt.Errorf("no selections")
	}

	showInputs := func(r *ui.Renderer) {
		if len(warnings) == 0 {
			return
		}
		renderWarningsSection(r, "warnings", warnings, false)
		r.Blank()
	}

	return apply(updated, showInputs, addedWorkspaceIDs)
}

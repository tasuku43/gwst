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
	"github.com/tasuku43/gwst/internal/app/rm"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/gitcmd"
	"github.com/tasuku43/gwst/internal/infra/output"
	"github.com/tasuku43/gwst/internal/ui"
)

func runWorkspaceAdd(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printAddHelp(os.Stdout)
		return nil
	}
	if len(args) > 2 {
		return fmt.Errorf("usage: gwst add [<WORKSPACE_ID>] [<repo>]")
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
			workspaceID, repoSpec, err = ui.PromptWorkspaceAndRepo("gwst add", workspaceChoices, repoChoices, workspaceID, repoSpec, theme, useColor)
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
	if err := rebuildManifest(ctx, rootDir); err != nil {
		return err
	}
	wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
	repos, _, _ := loadWorkspaceRepos(ctx, wsDir)
	renderer.Blank()
	renderer.Section("Result")
	description := loadWorkspaceDescription(wsDir)
	renderWorkspaceBlock(renderer, workspaceID, description, repos)
	renderSuggestions(renderer, useColor, []string{
		fmt.Sprintf("gwst open %s", workspaceID),
	})
	return nil
}

func runWorkspaceList(ctx context.Context, rootDir string, args []string) error {
	lsFlags := flag.NewFlagSet("ls", flag.ContinueOnError)
	lsFlags.SetOutput(os.Stdout)
	var showDetails bool
	var helpFlag bool
	lsFlags.BoolVar(&showDetails, "details", false, "show git status details")
	lsFlags.BoolVar(&helpFlag, "help", false, "show help")
	lsFlags.BoolVar(&helpFlag, "h", false, "show help")
	if err := lsFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printLsHelp(os.Stdout)
		return nil
	}
	if lsFlags.NArg() != 0 {
		return fmt.Errorf("usage: gwst ls [--details]")
	}
	entries, warnings, err := workspace.List(rootDir)
	if err != nil {
		return err
	}
	writeWorkspaceListText(ctx, rootDir, entries, warnings, showDetails)
	return nil
}

func runWorkspaceStatus(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printStatusHelp(os.Stdout)
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: gwst status [<WORKSPACE_ID>]")
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
		workspaceID, err = ui.PromptWorkspace("gwst status", workspaceChoices, theme, useColor)
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
		return fmt.Errorf("usage: gwst rm [<WORKSPACE_ID>]")
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
		selected, err = ui.PromptWorkspaceMultiSelectWithBlocked("gwst rm", removable, nil, theme, useColor)
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
		_, state := loadWorkspaceStatusForRemoval(ctx, rootDir, workspaceID)
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

		if err := rm.Remove(ctx, rootDir, workspaceID, state.Kind == workspace.WorkspaceStateDirty); err != nil {
			return err
		}
		if err := rebuildManifest(ctx, rootDir); err != nil {
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
		_, state := loadWorkspaceStatusForRemoval(ctx, rootDir, selectedID)
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
		if err := rm.Remove(ctx, rootDir, selectedID, state.Kind == workspace.WorkspaceStateDirty); err != nil {
			return err
		}
	}
	if err := rebuildManifest(ctx, rootDir); err != nil {
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	for _, selectedID := range selected {
		renderer.Bullet(fmt.Sprintf("%s removed", selectedID))
	}
	return nil
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

func buildWorkspaceChoiceWithDetails(ctx context.Context, entry workspace.Entry, repos []workspace.Repo, status workspace.StatusResult) ui.WorkspaceChoice {
	choice := ui.WorkspaceChoice{
		ID:          entry.WorkspaceID,
		Description: entry.Description,
	}
	statusByPath := make(map[string]workspace.RepoStatus, len(status.Repos))
	for _, repoStatus := range status.Repos {
		statusByPath[repoStatus.WorktreePath] = repoStatus
	}
	for _, repoEntry := range repos {
		name := formatRepoName(repoEntry.Alias, repoEntry.RepoKey)
		label := formatRepoLabel(name, repoEntry.Branch)
		prompt := ui.PromptChoice{
			Label: label,
			Value: displayRepoKey(repoEntry.RepoKey),
		}
		if repoStatus, ok := statusByPath[repoEntry.WorktreePath]; ok {
			prompt.Details = buildRepoStatusDetails(ctx, repoStatus)
		}
		choice.Repos = append(choice.Repos, prompt)
	}
	return choice
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

func loadWorkspaceStatusForRemoval(ctx context.Context, rootDir, workspaceID string) (workspace.StatusResult, workspace.WorkspaceState) {
	status, err := workspace.Status(ctx, rootDir, workspaceID)
	if err == nil {
		return status, workspace.StateFromStatus(status)
	}
	return workspace.StatusResult{
			WorkspaceID: workspaceID,
			Warnings:    []error{err},
		}, workspace.WorkspaceState{
			WorkspaceID: workspaceID,
			Kind:        workspace.WorkspaceStateUnknown,
			Warnings:    []error{err},
		}
}

func classifyWorkspaceRemoval(ctx context.Context, rootDir string, entries []workspace.Entry) ([]ui.WorkspaceChoice, []ui.BlockedChoice) {
	var removable []ui.WorkspaceChoice
	for _, entry := range entries {
		status, state := loadWorkspaceStatusForRemoval(ctx, rootDir, entry.WorkspaceID)
		choice := buildWorkspaceRemoveChoice(ctx, entry, status)
		choice.Warning, choice.WarningStrong = workspaceRemoveWarningLabel(state)
		removable = append(removable, choice)
	}
	return removable, nil
}

func buildWorkspaceRemoveChoice(ctx context.Context, entry workspace.Entry, status workspace.StatusResult) ui.WorkspaceChoice {
	choice := ui.WorkspaceChoice{
		ID:          entry.WorkspaceID,
		Description: entry.Description,
	}
	for _, repo := range status.Repos {
		name := strings.TrimSpace(repo.Alias)
		if name == "" {
			name = filepath.Base(repo.WorktreePath)
		}
		label := formatRepoLabel(name, repo.Branch)
		choice.Repos = append(choice.Repos, ui.PromptChoice{
			Label:   label,
			Value:   name,
			Details: buildRepoStatusDetails(ctx, repo),
		})
	}
	return choice
}

func buildRepoStatusDetails(ctx context.Context, repo workspace.RepoStatus) []string {
	if !repoNeedsStatusDetails(repo) {
		return nil
	}
	if repo.Error != nil {
		return []string{fmt.Sprintf("status error: %s", compactError(repo.Error))}
	}
	out, err := gitcmd.StatusShortBranch(ctx, repo.WorktreePath)
	if err != nil {
		return []string{fmt.Sprintf("status error: %s", compactError(err))}
	}
	return splitNonEmptyLines(out)
}

func repoNeedsStatusDetails(repo workspace.RepoStatus) bool {
	if repo.Error != nil {
		return true
	}
	if repo.Dirty {
		return true
	}
	if repo.Detached || repo.HeadMissing {
		return true
	}
	if strings.TrimSpace(repo.Upstream) == "" {
		return true
	}
	if repo.AheadCount > 0 {
		return true
	}
	return false
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

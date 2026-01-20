package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/apply"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/output"
	"github.com/tasuku43/gwst/internal/ui"
)

func runApply(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printApplyHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gwst apply")
	}

	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		return err
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	var warningLines []string
	for _, warn := range plan.Warnings {
		warningLines = append(warningLines, warn.Error())
	}
	if len(warningLines) > 0 {
		renderWarningsSection(renderer, "warnings", warningLines, false)
		renderer.Blank()
	}

	renderer.Section("Diff")
	if len(plan.Changes) == 0 {
		renderer.Bullet("no changes")
		return nil
	}
	renderPlanChanges(renderer, plan.Changes)

	destructive := planHasDestructiveChanges(plan)
	if destructive && noPrompt {
		return fmt.Errorf("destructive changes require confirmation")
	}
	if !noPrompt {
		renderer.Blank()
		label := "Apply changes? (default: No)"
		if destructive {
			label = "Apply destructive changes? (default: No)"
		}
		var confirm bool
		var err error
		if destructive {
			lines := buildApplyDestructiveConfirmLines(ctx, rootDir, plan, useColor, theme)
			if len(lines) > 0 {
				confirm, err = ui.PromptConfirmInlineWithRaw(label, lines, theme, useColor)
			} else {
				confirm, err = ui.PromptConfirmInline(label, theme, useColor)
			}
		} else {
			confirm, err = ui.PromptConfirmInline(label, theme, useColor)
		}
		if err != nil {
			return err
		}
		if !confirm {
			return nil
		}
	}

	renderer.Blank()
	renderer.Section("Steps")
	if err := apply.Apply(ctx, rootDir, plan, apply.Options{
		AllowDirty:       destructive,
		AllowStatusError: destructive,
		PrefetchTimeout:  defaultPrefetchTimeout,
		Step:             output.Step,
	}); err != nil {
		return err
	}
	if err := rebuildManifest(ctx, rootDir); err != nil {
		return err
	}

	renderer.Blank()
	renderer.Section("Result")
	renderer.Bullet("applied")
	return nil
}

func planHasDestructiveChanges(plan manifestplan.Result) bool {
	for _, change := range plan.Changes {
		switch change.Kind {
		case manifestplan.WorkspaceRemove:
			return true
		case manifestplan.WorkspaceUpdate:
			if hasDestructiveRepoChange(change.Repos) {
				return true
			}
		}
	}
	return false
}

func hasDestructiveRepoChange(changes []manifestplan.RepoChange) bool {
	for _, change := range changes {
		switch change.Kind {
		case manifestplan.RepoRemove, manifestplan.RepoUpdate:
			return true
		}
	}
	return false
}

func buildApplyDestructiveConfirmLines(ctx context.Context, rootDir string, plan manifestplan.Result, useColor bool, theme ui.Theme) []string {
	if len(plan.Changes) == 0 {
		return nil
	}

	type cachedStatus struct {
		status workspace.StatusResult
		state  workspace.WorkspaceState
	}
	statusCache := make(map[string]cachedStatus)
	getStatus := func(workspaceID string) cachedStatus {
		if cached, ok := statusCache[workspaceID]; ok {
			return cached
		}
		status, state := loadWorkspaceStatusForRemoval(ctx, rootDir, workspaceID)
		cached := cachedStatus{status: status, state: state}
		statusCache[workspaceID] = cached
		return cached
	}

	var choices []ui.WorkspaceChoice
	for _, change := range plan.Changes {
		switch change.Kind {
		case manifestplan.WorkspaceRemove:
			cached := getStatus(change.WorkspaceID)
			choice := buildApplyWorkspaceChoice(ctx, change.WorkspaceID, planWorkspaceDescription(plan, change.WorkspaceID), cached.status)
			choice.Warning, choice.WarningStrong = workspaceRemoveWarningLabel(cached.state)
			choices = append(choices, choice)
		case manifestplan.WorkspaceUpdate:
			if !hasDestructiveRepoChange(change.Repos) {
				continue
			}
			cached := getStatus(change.WorkspaceID)
			choice := buildApplyWorkspaceChoice(ctx, change.WorkspaceID, planWorkspaceDescription(plan, change.WorkspaceID), cached.status)
			choice.Warning, choice.WarningStrong = workspaceRemoveWarningLabel(cached.state)
			choices = append(choices, choice)
		}
	}
	if len(choices) == 0 {
		return nil
	}
	return ui.WorkspaceChoiceConfirmLines(choices, useColor, theme)
}

func planWorkspaceDescription(plan manifestplan.Result, workspaceID string) string {
	if ws, ok := plan.Actual.Workspaces[workspaceID]; ok {
		return ws.Description
	}
	if ws, ok := plan.Desired.Workspaces[workspaceID]; ok {
		return ws.Description
	}
	return ""
}

func buildApplyWorkspaceChoice(ctx context.Context, workspaceID, description string, status workspace.StatusResult) ui.WorkspaceChoice {
	choice := ui.WorkspaceChoice{
		ID:          workspaceID,
		Description: description,
	}
	for _, repoEntry := range status.Repos {
		name := strings.TrimSpace(repoEntry.Alias)
		if name == "" {
			name = filepath.Base(repoEntry.WorktreePath)
		}
		label := formatRepoLabel(name, repoEntry.Branch)
		choice.Repos = append(choice.Repos, ui.PromptChoice{
			Label:   label,
			Value:   name,
			Details: buildRepoStatusDetails(ctx, repoEntry),
		})
	}
	return choice
}

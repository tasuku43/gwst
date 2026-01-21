package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/apply"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
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

	renderer.Section("Plan")
	if len(plan.Changes) == 0 {
		renderer.Bullet("no changes")
		return nil
	}
	renderPlanChanges(ctx, rootDir, renderer, plan)

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
			confirm, err = ui.PromptConfirmInline(label, theme, useColor)
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
		case manifestplan.RepoRemove:
			return true
		case manifestplan.RepoUpdate:
			if isInPlaceBranchRename(change) {
				continue
			}
			return true
		}
	}
	return false
}

func isInPlaceBranchRename(change manifestplan.RepoChange) bool {
	if change.Kind != manifestplan.RepoUpdate {
		return false
	}
	if strings.TrimSpace(change.FromRepo) == "" || strings.TrimSpace(change.ToRepo) == "" {
		return false
	}
	if strings.TrimSpace(change.FromBranch) == "" || strings.TrimSpace(change.ToBranch) == "" {
		return false
	}
	if strings.TrimSpace(change.FromRepo) != strings.TrimSpace(change.ToRepo) {
		return false
	}
	return strings.TrimSpace(change.FromBranch) != strings.TrimSpace(change.ToBranch)
}

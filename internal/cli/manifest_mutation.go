package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/ui"
)

type manifestMutationHooks struct {
	ShowPrelude           func(*ui.Renderer)
	RenderNoApply         func(*ui.Renderer)
	RenderNoChanges       func(*ui.Renderer)
	RenderInfoBeforeApply func(r *ui.Renderer, plan manifestplan.Result, planOK bool)
}

type manifestMutationOptions struct {
	NoApply       bool
	NoPrompt      bool
	OriginalBytes []byte
	Hooks         manifestMutationHooks
}

func applyManifestMutation(ctx context.Context, rootDir string, updated manifest.File, opts manifestMutationOptions) error {
	if err := manifest.Save(rootDir, updated); err != nil {
		return err
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	if opts.Hooks.ShowPrelude != nil {
		opts.Hooks.ShowPrelude(renderer)
	}

	if opts.NoApply {
		if opts.Hooks.ShowPrelude != nil {
			renderer.Blank()
		}
		if opts.Hooks.RenderNoApply != nil {
			opts.Hooks.RenderNoApply(renderer)
		}
		return nil
	}

	plan, planErr := manifestplan.Plan(ctx, rootDir)
	planOK := planErr == nil
	if !planOK {
		return planErr
	}
	if planOK && len(plan.Changes) == 0 {
		if opts.Hooks.ShowPrelude != nil {
			renderer.Blank()
		}
		if opts.Hooks.RenderNoChanges != nil {
			opts.Hooks.RenderNoChanges(renderer)
		}
		return nil
	}

	if opts.Hooks.ShowPrelude != nil {
		renderer.Blank()
	}
	if opts.Hooks.RenderInfoBeforeApply != nil {
		opts.Hooks.RenderInfoBeforeApply(renderer, plan, planOK)
		renderer.Blank()
	}

	res, err := runApplyInternalWithPlan(ctx, rootDir, renderer, opts.NoPrompt, plan)
	if err != nil {
		return err
	}
	if res.Canceled || (res.HadChanges && !res.Confirmed) {
		if err := os.WriteFile(manifest.Path(rootDir), opts.OriginalBytes, 0o644); err != nil {
			return fmt.Errorf("restore %s: %w", manifest.FileName, err)
		}
		renderer.Blank()
		renderer.Section("Result")
		if res.Canceled {
			renderer.Bullet(fmt.Sprintf("%s rolled back (apply canceled)", manifest.FileName))
		} else {
			renderer.Bullet(fmt.Sprintf("%s rolled back (apply declined)", manifest.FileName))
		}
		return nil
	}
	return nil
}

func planIncludesChangesOutsideWorkspaceIDs(plan manifestplan.Result, workspaceIDs []string) bool {
	inScope := map[string]struct{}{}
	for _, id := range workspaceIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		inScope[id] = struct{}{}
	}
	if len(inScope) == 0 {
		return false
	}

	for _, change := range plan.Changes {
		id := strings.TrimSpace(change.WorkspaceID)
		if id == "" {
			continue
		}
		if _, ok := inScope[id]; ok {
			continue
		}
		return true
	}
	return false
}

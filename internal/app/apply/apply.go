package apply

import (
	"context"
	"fmt"
	"time"

	"github.com/tasuku43/gwst/internal/app/add"
	"github.com/tasuku43/gwst/internal/app/create"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
	"github.com/tasuku43/gwst/internal/app/remove_repo"
	"github.com/tasuku43/gwst/internal/app/rm"
	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/prefetcher"
)

type Options struct {
	AllowDirty       bool
	AllowStatusError bool
	PrefetchTimeout  time.Duration
	Step             func(text string)
}

func Apply(ctx context.Context, rootDir string, plan manifestplan.Result, opts Options) error {
	prefetch := prefetcher.New(opts.PrefetchTimeout)
	toPrefetch := collectRepoSpecs(plan)
	_, _ = prefetch.StartAll(ctx, rootDir, toPrefetch)

	for _, change := range plan.Changes {
		if change.Kind != manifestplan.WorkspaceRemove {
			continue
		}
		logStep(opts.Step, fmt.Sprintf("remove workspace %s", change.WorkspaceID))
		if err := rm.Remove(ctx, rootDir, change.WorkspaceID, opts.AllowDirty); err != nil {
			return err
		}
	}

	for _, change := range plan.Changes {
		if change.Kind != manifestplan.WorkspaceUpdate {
			continue
		}
		if err := applyRepoRemovals(ctx, rootDir, change, opts); err != nil {
			return err
		}
	}

	if err := prefetch.WaitAll(ctx, toPrefetch); err != nil {
		return err
	}

	for _, change := range plan.Changes {
		switch change.Kind {
		case manifestplan.WorkspaceAdd:
			if err := applyWorkspaceAdd(ctx, rootDir, plan.Desired, change, opts.Step); err != nil {
				return err
			}
		case manifestplan.WorkspaceUpdate:
			if err := applyRepoAdds(ctx, rootDir, change, opts.Step); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyWorkspaceAdd(ctx context.Context, rootDir string, desired manifest.File, change manifestplan.WorkspaceChange, step func(text string)) error {
	ws, ok := desired.Workspaces[change.WorkspaceID]
	if !ok {
		return fmt.Errorf("workspace not found in manifest: %s", change.WorkspaceID)
	}
	logStep(step, fmt.Sprintf("create workspace %s", change.WorkspaceID))
	_, err := create.CreateWorkspace(ctx, rootDir, change.WorkspaceID, workspace.Metadata{
		Description: ws.Description,
		Mode:        ws.Mode,
		PresetName:  ws.PresetName,
		SourceURL:   ws.SourceURL,
	})
	if err != nil {
		return err
	}
	for _, repoEntry := range ws.Repos {
		logStep(step, fmt.Sprintf("worktree add %s", repoEntry.Alias))
		if _, err := add.AddRepo(ctx, rootDir, change.WorkspaceID, repoEntry.RepoKey, repoEntry.Alias, repoEntry.Branch); err != nil {
			return err
		}
	}
	return nil
}

func applyRepoRemovals(ctx context.Context, rootDir string, change manifestplan.WorkspaceChange, opts Options) error {
	for _, repoChange := range change.Repos {
		switch repoChange.Kind {
		case manifestplan.RepoRemove, manifestplan.RepoUpdate:
			logStep(opts.Step, fmt.Sprintf("worktree remove %s", repoChange.Alias))
			if err := remove_repo.RemoveRepo(ctx, rootDir, change.WorkspaceID, repoChange.Alias, remove_repo.Options{
				AllowDirty:       opts.AllowDirty,
				AllowStatusError: opts.AllowStatusError,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyRepoAdds(ctx context.Context, rootDir string, change manifestplan.WorkspaceChange, step func(text string)) error {
	for _, repoChange := range change.Repos {
		switch repoChange.Kind {
		case manifestplan.RepoAdd:
			logStep(step, fmt.Sprintf("worktree add %s", repoChange.Alias))
			if _, err := add.AddRepo(ctx, rootDir, change.WorkspaceID, repoChange.ToRepo, repoChange.Alias, repoChange.ToBranch); err != nil {
				return err
			}
		case manifestplan.RepoUpdate:
			logStep(step, fmt.Sprintf("worktree add %s", repoChange.Alias))
			if _, err := add.AddRepo(ctx, rootDir, change.WorkspaceID, repoChange.ToRepo, repoChange.Alias, repoChange.ToBranch); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectRepoSpecs(plan manifestplan.Result) []string {
	unique := map[string]struct{}{}
	for _, change := range plan.Changes {
		switch change.Kind {
		case manifestplan.WorkspaceAdd:
			ws, ok := plan.Desired.Workspaces[change.WorkspaceID]
			if !ok {
				continue
			}
			for _, repoEntry := range ws.Repos {
				spec := repo.SpecFromKey(repoEntry.RepoKey)
				unique[spec] = struct{}{}
			}
		case manifestplan.WorkspaceUpdate:
			for _, repoChange := range change.Repos {
				switch repoChange.Kind {
				case manifestplan.RepoAdd, manifestplan.RepoUpdate:
					spec := repo.SpecFromKey(repoChange.ToRepo)
					unique[spec] = struct{}{}
				}
			}
		}
	}
	var specs []string
	for spec := range unique {
		specs = append(specs, spec)
	}
	return specs
}

func logStep(step func(text string), text string) {
	if step == nil {
		return
	}
	step(text)
}

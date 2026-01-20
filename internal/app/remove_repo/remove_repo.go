package remove_repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/gitcmd"
)

type Options struct {
	AllowDirty       bool
	AllowStatusError bool
}

func RemoveRepo(ctx context.Context, rootDir, workspaceID, alias string, opts Options) error {
	if strings.TrimSpace(alias) == "" {
		return fmt.Errorf("alias is required")
	}
	repos, _, err := workspace.ScanRepos(ctx, workspace.WorkspaceDir(rootDir, workspaceID))
	if err != nil {
		return err
	}
	var target *workspace.Repo
	for i := range repos {
		if repos[i].Alias == alias {
			target = &repos[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("repo not found in workspace %s: %s", workspaceID, alias)
	}

	status, err := workspace.Status(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}
	for _, repoStatus := range status.Repos {
		if repoStatus.Alias != alias {
			continue
		}
		if repoStatus.Error != nil {
			if !opts.AllowStatusError {
				return fmt.Errorf("check status for %s: %w", alias, repoStatus.Error)
			}
			break
		}
		if repoStatus.Dirty && !opts.AllowDirty {
			return fmt.Errorf("repo has dirty changes: %s", alias)
		}
	}

	if strings.TrimSpace(target.StorePath) == "" {
		return fmt.Errorf("missing store path for %s", alias)
	}
	force := opts.AllowDirty
	if err := gitcmd.WorktreeRemove(ctx, target.StorePath, target.WorktreePath, force); err != nil {
		return fmt.Errorf("remove worktree %q: %w", alias, err)
	}
	return nil
}

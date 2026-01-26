package workspace

import (
	"context"
	"fmt"
	"os"

	"github.com/tasuku43/gion/internal/infra/gitcmd"
	"github.com/tasuku43/gion/internal/infra/paths"
)

type RemoveOptions struct {
	AllowStatusError bool
	AllowDirty       bool
}

func Remove(ctx context.Context, rootDir, workspaceID string) error {
	return RemoveWithOptions(ctx, rootDir, workspaceID, RemoveOptions{})
}

func RemoveWithOptions(ctx context.Context, rootDir, workspaceID string, opts RemoveOptions) error {
	if workspaceID == "" {
		return fmt.Errorf("workspace id is required")
	}
	if rootDir == "" {
		return fmt.Errorf("root directory is required")
	}
	if err := validateWorkspaceID(ctx, workspaceID); err != nil {
		return err
	}

	wsDir := WorkspaceDir(rootDir, workspaceID)
	if exists, err := paths.DirExists(wsDir); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("workspace does not exist: %s", wsDir)
	}

	repos, warnings, err := ScanRepos(ctx, wsDir)
	if err != nil {
		return err
	}
	_ = warnings

	for _, repo := range repos {
		if repo.WorktreePath == "" {
			return fmt.Errorf("missing worktree path for alias %q", repo.Alias)
		}
		statusOut, statusErr := gitStatusPorcelain(ctx, repo.WorktreePath)
		if statusErr != nil {
			if !opts.AllowStatusError {
				return fmt.Errorf("check status for %q: %w", repo.Alias, statusErr)
			}
			continue
		}
		_, _, _, _, _, dirty, _, _, _, _, _, _ := parseStatusPorcelainV2(statusOut, "")
		if dirty {
			if !opts.AllowDirty {
				return fmt.Errorf("workspace has dirty changes: %s", repo.Alias)
			}
		}
	}

	for _, repo := range repos {
		if repo.StorePath == "" {
			continue
		}
		if repo.WorktreePath == "" {
			return fmt.Errorf("missing worktree path for alias %q", repo.Alias)
		}
		force := opts.AllowDirty
		if force {
			gitcmd.Logf("git worktree remove --force %s", repo.WorktreePath)
		} else {
			gitcmd.Logf("git worktree remove %s", repo.WorktreePath)
		}
		if err := gitcmd.WorktreeRemove(ctx, repo.StorePath, repo.WorktreePath, force); err != nil {
			return fmt.Errorf("remove worktree %q: %w", repo.Alias, err)
		}
	}

	if err := os.RemoveAll(wsDir); err != nil {
		return fmt.Errorf("remove workspace dir: %w", err)
	}

	return nil
}

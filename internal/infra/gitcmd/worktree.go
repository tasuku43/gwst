package gitcmd

import (
	"context"
	"fmt"
	"strings"
)

// WorktreePrune cleans up stale worktree metadata.
func WorktreePrune(ctx context.Context, dir string) error {
	res, err := Run(ctx, []string{"worktree", "prune"}, Options{Dir: dir})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return fmt.Errorf("git worktree prune failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return fmt.Errorf("git worktree prune failed: %w", err)
	}
	return nil
}

// WorktreeListPorcelain lists worktrees in porcelain format.
func WorktreeListPorcelain(ctx context.Context, dir string) (string, error) {
	res, err := Run(ctx, []string{"worktree", "list", "--porcelain"}, Options{Dir: dir})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return "", fmt.Errorf("git worktree list failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return "", fmt.Errorf("git worktree list failed: %w", err)
	}
	return res.Stdout, nil
}

// WorktreeAddExistingBranch adds a worktree for an existing branch.
func WorktreeAddExistingBranch(ctx context.Context, dir, path, branch string) error {
	res, err := Run(ctx, []string{"worktree", "add", path, branch}, Options{Dir: dir, ShowOutput: true})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return fmt.Errorf("git worktree add failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return fmt.Errorf("git worktree add failed: %w", err)
	}
	return nil
}

// WorktreeAddNewBranch adds a worktree with a new branch from baseRef.
func WorktreeAddNewBranch(ctx context.Context, dir, branch, path, baseRef string) error {
	res, err := Run(ctx, []string{"worktree", "add", "-b", branch, path, baseRef}, Options{Dir: dir, ShowOutput: true})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return fmt.Errorf("git worktree add failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return fmt.Errorf("git worktree add failed: %w", err)
	}
	return nil
}

// WorktreeAddTrackingBranch adds a worktree with a new branch tracking remoteName.
func WorktreeAddTrackingBranch(ctx context.Context, dir, branch, path, remoteName string) error {
	res, err := Run(ctx, []string{"worktree", "add", "-b", branch, "--track", path, remoteName}, Options{Dir: dir, ShowOutput: true})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return fmt.Errorf("git worktree add failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return fmt.Errorf("git worktree add failed: %w", err)
	}
	return nil
}

// WorktreeRemove removes a worktree.
func WorktreeRemove(ctx context.Context, dir, path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = []string{"worktree", "remove", "--force", path}
	}
	res, err := Run(ctx, args, Options{Dir: dir})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return fmt.Errorf("git worktree remove failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return fmt.Errorf("git worktree remove failed: %w", err)
	}
	return nil
}

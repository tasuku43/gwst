package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gws/internal/core/gitcmd"
	"github.com/tasuku43/gws/internal/domain/repo"
	"github.com/tasuku43/gws/internal/domain/repospec"
)

func Add(ctx context.Context, rootDir, workspaceID, repoSpec, alias string, fetch bool) (Repo, error) {
	return AddWithBranch(ctx, rootDir, workspaceID, repoSpec, alias, workspaceID, "", fetch)
}

func AddWithBranch(ctx context.Context, rootDir, workspaceID, repoSpec, alias, branch, baseRef string, fetch bool) (Repo, error) {
	if err := validateWorkspaceID(ctx, workspaceID); err != nil {
		return Repo{}, err
	}
	if err := validateBranchName(ctx, branch); err != nil {
		return Repo{}, err
	}
	if rootDir == "" {
		return Repo{}, fmt.Errorf("root directory is required")
	}

	wsDir := filepath.Join(rootDir, "workspaces", workspaceID)
	if exists, err := pathExists(wsDir); err != nil {
		return Repo{}, err
	} else if !exists {
		return Repo{}, fmt.Errorf("workspace does not exist: %s", wsDir)
	}

	spec, err := repospec.Normalize(repoSpec)
	if err != nil {
		return Repo{}, err
	}
	if alias == "" {
		alias = spec.Repo
	}
	if alias == "" {
		return Repo{}, fmt.Errorf("alias is required")
	}
	repos, _, err := ScanRepos(ctx, wsDir)
	if err != nil {
		return Repo{}, err
	}
	for _, existing := range repos {
		if existing.Alias == alias {
			return Repo{}, fmt.Errorf("alias already exists: %s", alias)
		}
		if spec.RepoKey != "" && existing.RepoKey == spec.RepoKey {
			return Repo{}, fmt.Errorf("repo already exists: %s", spec.RepoKey)
		}
	}

	store, err := repo.Open(ctx, rootDir, repoSpec, fetch)
	if err != nil {
		return Repo{}, err
	}

	worktreePath := filepath.Join(wsDir, alias)
	if exists, err := pathExists(worktreePath); err != nil {
		return Repo{}, err
	} else if exists {
		return Repo{}, fmt.Errorf("worktree already exists: %s", worktreePath)
	}

	if baseRef == "" {
		var err error
		baseRef, err = resolveBaseRef(ctx, store.StorePath)
		if err != nil {
			return Repo{}, err
		}
	}

	branchExists, err := branchExistsInStore(ctx, store.StorePath, branch)
	if err != nil {
		return Repo{}, err
	}

	if branchExists {
		gitcmd.Logf("git worktree add %s %s", worktreePath, branch)
		if _, err := gitcmd.Run(ctx, []string{"worktree", "add", worktreePath, branch}, gitcmd.Options{Dir: store.StorePath, ShowOutput: true}); err != nil {
			return Repo{}, err
		}
	} else {
		gitcmd.Logf("git worktree add -b %s %s %s", branch, worktreePath, baseRef)
		if _, err := gitcmd.Run(ctx, []string{"worktree", "add", "-b", branch, worktreePath, baseRef}, gitcmd.Options{Dir: store.StorePath, ShowOutput: true}); err != nil {
			return Repo{}, err
		}
	}

	repoEntry := Repo{
		Alias:        alias,
		RepoSpec:     repoSpec,
		RepoKey:      spec.RepoKey,
		StorePath:    store.StorePath,
		WorktreePath: worktreePath,
		Branch:       branch,
	}

	return repoEntry, nil
}

func resolveBaseRef(ctx context.Context, storePath string) (string, error) {
	if storePath == "" {
		return "", fmt.Errorf("store path is required")
	}

	remoteHead, remoteErr := detectDefaultRemoteRef(ctx, storePath)
	if remoteErr == nil && remoteHead != "" {
		return remoteHead, nil
	}

	localHead, localErr := detectLocalHeadRef(ctx, storePath)
	if localErr == nil && localHead != "" {
		return localHead, nil
	}

	for _, candidate := range []string{"main", "master", "develop"} {
		exists, err := localRefExists(ctx, storePath, candidate)
		if err != nil {
			return "", err
		}
		if exists {
			return fmt.Sprintf("refs/heads/%s", candidate), nil
		}
	}

	for _, candidate := range []string{"origin/main", "origin/master", "origin/develop"} {
		exists, err := remoteRefExists(ctx, storePath, candidate)
		if err != nil {
			return "", err
		}
		if exists {
			return candidate, nil
		}
	}

	if remoteErr != nil {
		return "", remoteErr
	}
	if localErr != nil {
		return "", localErr
	}
	return "", fmt.Errorf("cannot detect default base ref")
}

func detectLocalHeadRef(ctx context.Context, storePath string) (string, error) {
	res, err := gitcmd.Run(ctx, []string{"symbolic-ref", "--quiet", "HEAD"}, gitcmd.Options{Dir: storePath})
	if err != nil {
		if res.ExitCode == 1 {
			return "", nil
		}
		if strings.TrimSpace(res.Stderr) != "" {
			return "", fmt.Errorf("git symbolic-ref HEAD failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return "", err
	}
	ref := strings.TrimSpace(res.Stdout)
	if ref == "" {
		return "", nil
	}
	return ref, nil
}

func detectDefaultRemoteRef(ctx context.Context, storePath string) (string, error) {
	res, err := gitcmd.Run(ctx, []string{"symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"}, gitcmd.Options{Dir: storePath})
	if err != nil {
		if res.ExitCode == 1 {
			return "", nil
		}
		if strings.TrimSpace(res.Stderr) != "" {
			return "", fmt.Errorf("git symbolic-ref origin/HEAD failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return "", err
	}
	ref := strings.TrimSpace(res.Stdout)
	if !strings.HasPrefix(ref, "refs/remotes/") {
		return "", nil
	}
	return strings.TrimPrefix(ref, "refs/remotes/"), nil
}

func localRefExists(ctx context.Context, storePath, name string) (bool, error) {
	fullRef := fmt.Sprintf("refs/heads/%s", name)
	return refExists(ctx, storePath, fullRef)
}

func remoteRefExists(ctx context.Context, storePath, ref string) (bool, error) {
	fullRef := fmt.Sprintf("refs/remotes/%s", ref)
	return refExists(ctx, storePath, fullRef)
}

func refExists(ctx context.Context, storePath, fullRef string) (bool, error) {
	res, err := gitcmd.Run(ctx, []string{"show-ref", "--verify", fullRef}, gitcmd.Options{Dir: storePath})
	if err == nil {
		return true, nil
	}
	if res.ExitCode == 1 {
		return false, nil
	}
	if res.ExitCode == 128 && strings.Contains(res.Stderr, "not a valid ref") {
		return false, nil
	}
	if strings.TrimSpace(res.Stderr) != "" {
		return false, fmt.Errorf("git show-ref failed: %w: %s", err, strings.TrimSpace(res.Stderr))
	}
	return false, err
}

func branchExistsInStore(ctx context.Context, storePath, branch string) (bool, error) {
	ref := fmt.Sprintf("refs/heads/%s", branch)
	res, err := gitcmd.Run(ctx, []string{"show-ref", "--verify", ref}, gitcmd.Options{Dir: storePath})
	if err == nil {
		return true, nil
	}
	if res.ExitCode == 1 {
		return false, nil
	}
	if res.ExitCode == 128 && strings.Contains(res.Stderr, "not a valid ref") {
		return false, nil
	}
	return false, err
}

func validateBranchName(ctx context.Context, branch string) error {
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("branch is required")
	}
	_, err := gitcmd.Run(ctx, []string{"check-ref-format", "--branch", branch}, gitcmd.Options{})
	if err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}
	return nil
}

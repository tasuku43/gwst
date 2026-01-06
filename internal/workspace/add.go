package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tasuku43/gws/internal/config"
	"github.com/tasuku43/gws/internal/gitcmd"
	"github.com/tasuku43/gws/internal/repo"
	"github.com/tasuku43/gws/internal/repospec"
)

func Add(ctx context.Context, rootDir, workspaceID, repoSpec, alias string, cfg config.Config, fetch bool) (Repo, error) {
	if err := validateWorkspaceID(ctx, workspaceID); err != nil {
		return Repo{}, err
	}
	if rootDir == "" {
		return Repo{}, fmt.Errorf("root directory is required")
	}

	wsDir := filepath.Join(rootDir, "ws", workspaceID)
	if exists, err := pathExists(wsDir); err != nil {
		return Repo{}, err
	} else if !exists {
		return Repo{}, fmt.Errorf("workspace does not exist: %s", wsDir)
	}

	manifestPath := filepath.Join(wsDir, manifestDirName, manifestFileName)
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return Repo{}, err
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
	if err := ensureRepoNotRegistered(manifest, alias, spec.RepoKey); err != nil {
		return Repo{}, err
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

	branch := workspaceID
	baseRef, err := resolveBaseRef(ctx, store.StorePath, cfg)
	if err != nil {
		return Repo{}, err
	}

	branchExists, err := branchExistsInStore(ctx, store.StorePath, branch)
	if err != nil {
		return Repo{}, err
	}

	var createdBranch bool
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
		createdBranch = true
	}

	repoEntry := Repo{
		Alias:         alias,
		RepoSpec:      repoSpec,
		RepoKey:       spec.RepoKey,
		StorePath:     store.StorePath,
		WorktreePath:  worktreePath,
		Branch:        branch,
		BaseRef:       baseRef,
		CreatedBranch: createdBranch,
	}

	AddRepo(&manifest, repoEntry)
	TouchLastUsed(&manifest, time.Now())
	if err := WriteManifest(manifestPath, manifest); err != nil {
		return Repo{}, err
	}

	return repoEntry, nil
}

func resolveBaseRef(ctx context.Context, storePath string, cfg config.Config) (string, error) {
	if storePath == "" {
		return "", fmt.Errorf("store path is required")
	}

	if base := strings.TrimSpace(cfg.Defaults.BaseRef); base != "" {
		return base, nil
	}

	localHead, err := detectLocalHeadRef(ctx, storePath)
	if err == nil && localHead != "" {
		return localHead, nil
	}

	remoteHead, err := detectDefaultRemoteRef(ctx, storePath)
	if err == nil && remoteHead != "" {
		return remoteHead, nil
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

	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("cannot detect default base ref; set defaults.base_ref")
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

func ensureRepoNotRegistered(manifest Manifest, alias, repoKey string) error {
	for _, existing := range manifest.Repos {
		if existing.Alias == alias {
			return fmt.Errorf("alias already exists: %s", alias)
		}
		if repoKey != "" && existing.RepoKey == repoKey {
			return fmt.Errorf("repo already registered: %s", repoKey)
		}
	}
	return nil
}

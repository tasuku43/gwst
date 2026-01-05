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

func Add(ctx context.Context, rootDir, workspaceID, repoSpec, alias string, cfg config.Config) (Repo, error) {
	if alias == "" {
		return Repo{}, fmt.Errorf("alias is required")
	}
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
	if err := ensureRepoNotRegistered(manifest, alias, spec.RepoKey); err != nil {
		return Repo{}, err
	}

	store, err := repo.Open(ctx, rootDir, repoSpec)
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
	baseRef := strings.TrimSpace(cfg.Defaults.BaseRef)
	if baseRef == "" {
		baseRef = "origin/main"
	}

	branchExists, err := branchExistsInStore(ctx, store.StorePath, branch)
	if err != nil {
		return Repo{}, err
	}

	var createdBranch bool
	if branchExists {
		if _, err := gitcmd.Run(ctx, []string{"worktree", "add", worktreePath, branch}, gitcmd.Options{Dir: store.StorePath}); err != nil {
			return Repo{}, err
		}
	} else {
		if _, err := gitcmd.Run(ctx, []string{"worktree", "add", "-b", branch, worktreePath, baseRef}, gitcmd.Options{Dir: store.StorePath}); err != nil {
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

func branchExistsInStore(ctx context.Context, storePath, branch string) (bool, error) {
	ref := fmt.Sprintf("refs/heads/%s", branch)
	res, err := gitcmd.Run(ctx, []string{"show-ref", "--verify", ref}, gitcmd.Options{Dir: storePath})
	if err == nil {
		return true, nil
	}
	if res.ExitCode == 1 {
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

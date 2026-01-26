package add

import (
	"context"
	"fmt"
	"strings"

	"github.com/tasuku43/gion/internal/domain/repo"
	"github.com/tasuku43/gion/internal/domain/workspace"
	"github.com/tasuku43/gion/internal/infra/gitcmd"
)

func AddRepo(ctx context.Context, rootDir, workspaceID, repoKey, alias, branch, baseRef string, fetch bool) (workspace.Repo, bool, string, error) {
	repoSpec := repo.SpecFromKey(repoKey)
	_, exists, err := repo.Exists(rootDir, repoSpec)
	if err != nil {
		return workspace.Repo{}, false, "", err
	}
	if !exists {
		if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
			return workspace.Repo{}, false, "", err
		}
	}

	store, err := repo.Open(ctx, rootDir, repoSpec, false)
	if err != nil {
		return workspace.Repo{}, false, "", err
	}

	_, localBranchExists, err := gitcmd.ShowRef(ctx, store.StorePath, fmt.Sprintf("refs/heads/%s", branch))
	if err != nil {
		return workspace.Repo{}, false, "", err
	}
	_, remoteBranchExists, err := gitcmd.ShowRef(ctx, store.StorePath, fmt.Sprintf("refs/remotes/origin/%s", branch))
	if err != nil {
		return workspace.Repo{}, false, "", err
	}

	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" {
		baseRef, err = workspace.ResolveBaseRef(ctx, store.StorePath)
		if err != nil {
			return workspace.Repo{}, false, "", err
		}
	}

	added, err := workspace.AddWithBranch(ctx, rootDir, workspaceID, repoSpec, alias, branch, baseRef, fetch)
	if err != nil {
		return workspace.Repo{}, false, "", err
	}

	createdNewBranch := !(localBranchExists || remoteBranchExists)
	baseBranchForMetadata := ""
	if createdNewBranch && strings.HasPrefix(baseRef, "origin/") {
		baseBranchForMetadata = baseRef
	}
	return added, createdNewBranch, baseBranchForMetadata, nil
}

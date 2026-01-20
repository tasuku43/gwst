package add

import (
	"context"

	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
)

func AddRepo(ctx context.Context, rootDir, workspaceID, repoKey, alias, branch string) (workspace.Repo, error) {
	repoSpec := repo.SpecFromKey(repoKey)
	_, exists, err := repo.Exists(rootDir, repoSpec)
	if err != nil {
		return workspace.Repo{}, err
	}
	if !exists {
		if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
			return workspace.Repo{}, err
		}
	}
	return workspace.AddWithBranch(ctx, rootDir, workspaceID, repoSpec, alias, branch, "", true)
}

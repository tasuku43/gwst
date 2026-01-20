package rm

import (
	"context"

	"github.com/tasuku43/gwst/internal/domain/workspace"
)

func Remove(ctx context.Context, rootDir, workspaceID string, allowDirty bool) error {
	return workspace.RemoveWithOptions(ctx, rootDir, workspaceID, workspace.RemoveOptions{
		AllowStatusError: true,
		AllowDirty:       allowDirty,
	})
}

package workspace

import (
	"path/filepath"

	"github.com/tasuku43/gwst/internal/infra/paths"
)

// WorkspacesRoot returns the path to the workspaces root directory.
func WorkspacesRoot(rootDir string) string {
	return paths.WorkspacesRoot(rootDir)
}

// WorkspaceDir returns the path to a specific workspace directory.
func WorkspaceDir(rootDir, workspaceID string) string {
	return filepath.Join(paths.WorkspacesRoot(rootDir), workspaceID)
}

// WorktreePath returns the path to a repo worktree under a workspace.
func WorktreePath(rootDir, workspaceID, alias string) string {
	return filepath.Join(WorkspaceDir(rootDir, workspaceID), alias)
}

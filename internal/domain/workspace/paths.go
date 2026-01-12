package workspace

import (
	"path/filepath"

	"github.com/tasuku43/gws/internal/core/paths"
)

// WorkspacesRoot returns the path to the workspaces root directory.
func WorkspacesRoot(rootDir string) string {
	return paths.WorkspacesRoot(rootDir)
}

// WorkspaceDir returns the path to a specific workspace directory.
func WorkspaceDir(rootDir, workspaceID string) string {
	return filepath.Join(paths.WorkspacesRoot(rootDir), workspaceID)
}

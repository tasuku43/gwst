package paths

import "path/filepath"

// BareRoot returns the path to the bare repo store root.
func BareRoot(rootDir string) string {
	return filepath.Join(rootDir, "bare")
}

// WorkspacesRoot returns the path to the workspaces root.
func WorkspacesRoot(rootDir string) string {
	return filepath.Join(rootDir, "workspaces")
}

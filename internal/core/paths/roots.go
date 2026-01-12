package paths

import "path/filepath"

// BareRoot returns the path to the bare repo store root.
func BareRoot(rootDir string) string {
	return filepath.Join(rootDir, "bare")
}

// SrcRoot returns the path to the src working tree root.
func SrcRoot(rootDir string) string {
	return filepath.Join(rootDir, "src")
}

// WorkspacesRoot returns the path to the workspaces root.
func WorkspacesRoot(rootDir string) string {
	return filepath.Join(rootDir, "workspaces")
}

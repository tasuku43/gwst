package repo

import (
	"path/filepath"

	"github.com/tasuku43/gws/internal/core/paths"
	"github.com/tasuku43/gws/internal/domain/repospec"
)

// StorePath returns the path to the bare repo store for the spec.
func StorePath(rootDir string, spec repospec.Spec) string {
	return filepath.Join(paths.BareRoot(rootDir), spec.Host, spec.Owner, spec.Repo+".git")
}

// SrcPath returns the path to the working tree for the spec.
func SrcPath(rootDir string, spec repospec.Spec) string {
	return filepath.Join(paths.SrcRoot(rootDir), spec.Host, spec.Owner, spec.Repo)
}

package repo

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gwst/internal/core/paths"
	"github.com/tasuku43/gwst/internal/domain/repospec"
)

// Spec is the normalized repo specification.
type Spec = repospec.Spec

// StorePath returns the path to the bare repo store for the spec.
func StorePath(rootDir string, spec repospec.Spec) string {
	return filepath.Join(paths.BareRoot(rootDir), spec.Host, spec.Owner, spec.Repo+".git")
}

// Normalize trims and validates a repo spec, returning the spec and trimmed input.
func Normalize(input string) (repospec.Spec, string, error) {
	trimmed := strings.TrimSpace(input)
	spec, err := repospec.Normalize(trimmed)
	if err != nil {
		return repospec.Spec{}, "", err
	}
	return spec, trimmed, nil
}

// DisplaySpec returns a normalized display string for a repo spec.
func DisplaySpec(input string) string {
	spec, ok := normalizeForDisplay(input)
	if !ok {
		return strings.TrimSpace(input)
	}
	return fmt.Sprintf("git@%s:%s/%s.git", spec.Host, spec.Owner, spec.Repo)
}

// DisplayName returns the repo name for display.
func DisplayName(input string) string {
	spec, ok := normalizeForDisplay(input)
	if !ok || spec.Repo == "" {
		return strings.TrimSpace(input)
	}
	return spec.Repo
}

func normalizeForDisplay(input string) (repospec.Spec, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return repospec.Spec{}, false
	}
	spec, err := repospec.Normalize(trimmed)
	if err != nil {
		return repospec.Spec{}, false
	}
	return spec, true
}

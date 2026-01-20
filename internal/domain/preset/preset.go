package preset

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/tasuku43/gwst/internal/domain/manifest"
)

type File = manifest.File
type Preset = manifest.Preset

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func Load(rootDir string) (File, error) {
	return manifest.Load(rootDir)
}

func Names(file File) []string {
	var names []string
	for name := range file.Presets {
		if strings.TrimSpace(name) == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateName checks preset name rules.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("preset name is required")
	}
	if !namePattern.MatchString(trimmed) {
		return fmt.Errorf("invalid preset name: %s", name)
	}
	return nil
}

// NormalizeRepos trims and de-duplicates repo specs while preserving order.
func NormalizeRepos(repos []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, repo := range repos {
		trimmed := strings.TrimSpace(repo)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func Save(rootDir string, file File) error {
	return manifest.Save(rootDir, file)
}

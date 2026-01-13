package repo

import (
	"os"
	"path/filepath"

	"github.com/tasuku43/gws/internal/core/paths"
)

// ListSrc returns git repo directories under <root>/src.
func ListSrc(rootDir string) ([]string, []error, error) {
	srcRoot := paths.SrcRoot(rootDir)
	exists, err := paths.DirExists(srcRoot)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, nil
	}

	var entries []string
	var warnings []error

	err = filepath.WalkDir(srcRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, walkErr)
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		if path == srcRoot {
			return nil
		}

		gitDir := filepath.Join(path, ".git")
		isRepo, err := paths.DirExists(gitDir)
		if err != nil {
			warnings = append(warnings, err)
			return nil
		}
		if !isRepo {
			return nil
		}

		entries = append(entries, path)
		return filepath.SkipDir
	})
	if err != nil {
		return nil, warnings, err
	}

	return entries, warnings, nil
}

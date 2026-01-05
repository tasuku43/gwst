package src

import (
	"fmt"
	"os"
	"path/filepath"
)

type Entry struct {
	RepoKey string
	Path    string
}

func List(rootDir string) ([]Entry, []error, error) {
	srcRoot := filepath.Join(rootDir, "src")
	info, err := os.Stat(srcRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("src path is not a directory: %s", srcRoot)
	}

	var entries []Entry
	var warnings []error

	err = filepath.WalkDir(srcRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, walkErr)
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path == srcRoot {
			return nil
		}
		if !hasGitDir(path) {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			warnings = append(warnings, err)
			return nil
		}
		repoKey := filepath.ToSlash(rel)
		entries = append(entries, Entry{
			RepoKey: repoKey,
			Path:    path,
		})
		return filepath.SkipDir
	})
	if err != nil {
		return nil, warnings, err
	}

	return entries, warnings, nil
}

func hasGitDir(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

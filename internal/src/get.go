package src

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gws/internal/gitcmd"
	"github.com/tasuku43/gws/internal/repo"
	"github.com/tasuku43/gws/internal/repospec"
)

type Result struct {
	RepoKey string
	Path    string
}

func Get(ctx context.Context, rootDir, repoSpec string) (Result, error) {
	if rootDir == "" {
		return Result{}, fmt.Errorf("root directory is required")
	}
	if strings.TrimSpace(repoSpec) == "" {
		return Result{}, fmt.Errorf("repo is required")
	}

	spec, err := repospec.Normalize(repoSpec)
	if err != nil {
		return Result{}, err
	}
	srcPath := filepath.Join(rootDir, "src", spec.Host, spec.Owner, spec.Repo)

	store, err := repo.Open(ctx, rootDir, repoSpec)
	if err != nil {
		return Result{}, err
	}

	if exists, err := pathExists(srcPath); err != nil {
		return Result{}, err
	} else if exists {
		if _, err := gitcmd.Run(ctx, []string{"fetch", "--prune"}, gitcmd.Options{Dir: srcPath}); err != nil {
			return Result{}, err
		}
		return Result{RepoKey: spec.RepoKey, Path: srcPath}, nil
	}

	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("create src dir: %w", err)
	}
	if _, err := gitcmd.Run(ctx, []string{"clone", store.StorePath, srcPath}, gitcmd.Options{}); err != nil {
		return Result{}, err
	}
	_, _ = gitcmd.Run(ctx, []string{"remote", "set-url", "origin", repoSpec}, gitcmd.Options{Dir: srcPath})

	return Result{RepoKey: spec.RepoKey, Path: srcPath}, nil
}

func pathExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("path is not a directory: %s", path)
	}
	return true, nil
}

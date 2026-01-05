package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tasuku43/gws/internal/gitcmd"
	"github.com/tasuku43/gws/internal/repospec"
)

type Store struct {
	RepoKey   string
	StorePath string
	RemoteURL string
}

func Get(ctx context.Context, rootDir string, repo string) (Store, error) {
	spec, err := repospec.Normalize(repo)
	if err != nil {
		return Store{}, err
	}

	storePath := filepath.Join(rootDir, "bare", spec.Host, spec.Owner, spec.Repo+".git")

	exists, err := pathExists(storePath)
	if err != nil {
		return Store{}, err
	}

	if !exists {
		if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
			return Store{}, fmt.Errorf("create repo store dir: %w", err)
		}
		if _, err := gitcmd.Run(ctx, []string{"clone", "--bare", repo, storePath}, gitcmd.Options{}); err != nil {
			return Store{}, err
		}
	} else {
		if _, err := gitcmd.Run(ctx, []string{"fetch", "--prune"}, gitcmd.Options{Dir: storePath}); err != nil {
			return Store{}, err
		}
	}

	if err := ensureSrc(ctx, rootDir, spec, storePath, repo); err != nil {
		return Store{}, err
	}

	return Store{
		RepoKey:   spec.RepoKey,
		StorePath: storePath,
		RemoteURL: repo,
	}, nil
}

func Open(ctx context.Context, rootDir string, repo string) (Store, error) {
	spec, err := repospec.Normalize(repo)
	if err != nil {
		return Store{}, err
	}

	storePath := filepath.Join(rootDir, "bare", spec.Host, spec.Owner, spec.Repo+".git")

	exists, err := pathExists(storePath)
	if err != nil {
		return Store{}, err
	}
	if !exists {
		return Store{}, fmt.Errorf("repo store not found, run: gws repo get %s", repo)
	}

	if _, err := gitcmd.Run(ctx, []string{"fetch", "--prune"}, gitcmd.Options{Dir: storePath}); err != nil {
		return Store{}, err
	}

	return Store{
		RepoKey:   spec.RepoKey,
		StorePath: storePath,
		RemoteURL: repo,
	}, nil
}

func ensureSrc(ctx context.Context, rootDir string, spec repospec.Spec, storePath, repoSpec string) error {
	srcPath := filepath.Join(rootDir, "src", spec.Host, spec.Owner, spec.Repo)
	if exists, err := pathExists(srcPath); err != nil {
		return err
	} else if exists {
		if _, err := gitcmd.Run(ctx, []string{"fetch", "--prune"}, gitcmd.Options{Dir: srcPath}); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		return fmt.Errorf("create src dir: %w", err)
	}
	if _, err := gitcmd.Run(ctx, []string{"clone", storePath, srcPath}, gitcmd.Options{}); err != nil {
		return err
	}
	_, _ = gitcmd.Run(ctx, []string{"remote", "set-url", "origin", repoSpec}, gitcmd.Options{Dir: srcPath})
	return nil
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

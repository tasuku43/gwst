package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gws/internal/core/gitcmd"
	"github.com/tasuku43/gws/internal/domain/repospec"
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
	remoteURL := strings.TrimSpace(repo)

	storePath := storePathForSpec(rootDir, spec)

	exists, err := pathExists(storePath)
	if err != nil {
		return Store{}, err
	}

	if !exists {
		if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
			return Store{}, fmt.Errorf("create repo store dir: %w", err)
		}
		gitcmd.Logf("git clone --bare %s %s", remoteURL, storePath)
		if _, err := gitcmd.Run(ctx, []string{"clone", "--bare", remoteURL, storePath}, gitcmd.Options{}); err != nil {
			return Store{}, err
		}
	}

	if err := normalizeStore(ctx, storePath, spec.RepoKey, false); err != nil {
		return Store{}, err
	}

	if err := ensureSrc(ctx, rootDir, spec, storePath, remoteURL, false); err != nil {
		return Store{}, err
	}

	return Store{
		RepoKey:   spec.RepoKey,
		StorePath: storePath,
		RemoteURL: remoteURL,
	}, nil
}

func Open(ctx context.Context, rootDir string, repo string, fetch bool) (Store, error) {
	spec, err := repospec.Normalize(repo)
	if err != nil {
		return Store{}, err
	}
	remoteURL := strings.TrimSpace(repo)

	storePath := storePathForSpec(rootDir, spec)

	exists, err := pathExists(storePath)
	if err != nil {
		return Store{}, err
	}
	if !exists {
		return Store{}, fmt.Errorf("repo store not found, run: gws repo get %s", repo)
	}

	if err := normalizeStore(ctx, storePath, spec.RepoKey, fetch); err != nil {
		return Store{}, err
	}

	return Store{
		RepoKey:   spec.RepoKey,
		StorePath: storePath,
		RemoteURL: remoteURL,
	}, nil
}

func ensureSrc(ctx context.Context, rootDir string, spec repospec.Spec, storePath, remoteURL string, fetch bool) error {
	srcPath := filepath.Join(rootDir, "src", spec.Host, spec.Owner, spec.Repo)
	if exists, err := pathExists(srcPath); err != nil {
		return err
	} else if exists {
		if fetch {
			gitcmd.Logf("git fetch --prune")
			if _, err := gitcmd.Run(ctx, []string{"fetch", "--prune"}, gitcmd.Options{Dir: srcPath}); err != nil {
				return err
			}
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		return fmt.Errorf("create src dir: %w", err)
	}
	gitcmd.Logf("git clone %s %s", storePath, srcPath)
	if _, err := gitcmd.Run(ctx, []string{"clone", storePath, srcPath}, gitcmd.Options{}); err != nil {
		return err
	}
	_, _ = gitcmd.Run(ctx, []string{"remote", "set-url", "origin", remoteURL}, gitcmd.Options{Dir: srcPath})
	return nil
}

func Exists(rootDir, repo string) (string, bool, error) {
	spec, err := repospec.Normalize(repo)
	if err != nil {
		return "", false, err
	}
	storePath := storePathForSpec(rootDir, spec)
	exists, err := pathExists(storePath)
	if err != nil {
		return "", false, err
	}
	return storePath, exists, nil
}

func storePathForSpec(rootDir string, spec repospec.Spec) string {
	return filepath.Join(rootDir, "bare", spec.Host, spec.Owner, spec.Repo+".git")
}

func normalizeStore(ctx context.Context, storePath, display string, fetch bool) error {
	if _, err := gitcmd.Run(ctx, []string{"config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"}, gitcmd.Options{Dir: storePath}); err != nil {
		return err
	}
	defaultBranch, remoteHash, err := defaultBranchFromRemote(ctx, storePath)
	if err != nil {
		return err
	}
	if defaultBranch != "" {
		_, _ = gitcmd.Run(ctx, []string{"symbolic-ref", "refs/remotes/origin/HEAD", fmt.Sprintf("refs/remotes/origin/%s", defaultBranch)}, gitcmd.Options{Dir: storePath})
	}

	needsFetch := false
	localRemote, err := localRemoteHash(ctx, storePath, defaultBranch)
	if err != nil {
		return err
	}
	remoteTrackingMissing := localRemote == ""
	localHash := localRemote
	if localHash == "" {
		localHash, err = localHeadHash(ctx, storePath, defaultBranch)
		if err != nil {
			return err
		}
	}

	if fetch {
		needsFetch = true
	} else if remoteTrackingMissing {
		needsFetch = true
	} else if defaultBranch != "" && remoteHash != "" && localHash != "" && localHash != remoteHash {
		needsFetch = true
	}

	if needsFetch {
		gitcmd.Logf("git fetch --prune")
		if _, err := gitcmd.Run(ctx, []string{"fetch", "--prune"}, gitcmd.Options{Dir: storePath}); err != nil {
			return err
		}
	}
	return pruneLocalHeads(ctx, storePath, defaultBranch)
}

func defaultBranchFromRemote(ctx context.Context, storePath string) (string, string, error) {
	res, err := gitcmd.Run(ctx, []string{"ls-remote", "--symref", "origin", "HEAD"}, gitcmd.Options{Dir: storePath})
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	var branch string
	var hash string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "ref: ") && strings.HasSuffix(line, "\tHEAD") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ref := parts[1]
				ref = strings.TrimPrefix(ref, "refs/heads/")
				if ref != "" {
					branch = ref
				}
			}
			continue
		}
		if strings.HasSuffix(line, "\tHEAD") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				hash = fields[0]
			}
		}
	}
	return branch, hash, nil
}

func localRemoteHash(ctx context.Context, storePath, branch string) (string, error) {
	ref := fmt.Sprintf("refs/remotes/origin/%s", branch)
	res, err := gitcmd.Run(ctx, []string{"show-ref", "--verify", ref}, gitcmd.Options{Dir: storePath})
	if err == nil {
		fields := strings.Fields(strings.TrimSpace(res.Stdout))
		if len(fields) >= 1 {
			return fields[0], nil
		}
		return "", nil
	}
	if res.ExitCode == 1 || (res.ExitCode == 128 && strings.Contains(res.Stderr, "not a valid ref")) {
		return "", nil
	}
	return "", err
}

func localHeadHash(ctx context.Context, storePath, branch string) (string, error) {
	ref := fmt.Sprintf("refs/heads/%s", branch)
	res, err := gitcmd.Run(ctx, []string{"show-ref", "--verify", ref}, gitcmd.Options{Dir: storePath})
	if err == nil {
		fields := strings.Fields(strings.TrimSpace(res.Stdout))
		if len(fields) >= 1 {
			return fields[0], nil
		}
		return "", nil
	}
	if res.ExitCode == 1 || (res.ExitCode == 128 && strings.Contains(res.Stderr, "not a valid ref")) {
		return "", nil
	}
	return "", err
}

func pruneLocalHeads(ctx context.Context, storePath, keepBranch string) error {
	worktreeBranches, _ := worktreeBranchNames(ctx, storePath)
	res, err := gitcmd.Run(ctx, []string{"show-ref", "--heads"}, gitcmd.Options{Dir: storePath})
	if err != nil && res.ExitCode != 1 {
		return err
	}
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		ref := parts[1]
		if !strings.HasPrefix(ref, "refs/heads/") {
			continue
		}
		name := strings.TrimPrefix(ref, "refs/heads/")
		if name == keepBranch {
			continue
		}
		if _, ok := worktreeBranches[name]; ok {
			continue
		}
		_, _ = gitcmd.Run(ctx, []string{"update-ref", "-d", ref}, gitcmd.Options{Dir: storePath})
	}
	return nil
}

func worktreeBranchNames(ctx context.Context, storePath string) (map[string]struct{}, error) {
	res, err := gitcmd.Run(ctx, []string{"worktree", "list", "--porcelain"}, gitcmd.Options{Dir: storePath})
	if err != nil {
		return nil, err
	}
	branches := make(map[string]struct{})
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "branch ") {
			continue
		}
		ref := strings.TrimSpace(strings.TrimPrefix(line, "branch "))
		if strings.HasPrefix(ref, "refs/heads/") {
			name := strings.TrimPrefix(ref, "refs/heads/")
			if name != "" {
				branches[name] = struct{}{}
			}
		}
	}
	return branches, nil
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

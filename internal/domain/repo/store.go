package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tasuku43/gws/internal/core/gitcmd"
	"github.com/tasuku43/gws/internal/core/paths"
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

	exists, err := paths.DirExists(storePath)
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

	exists, err := paths.DirExists(storePath)
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
	srcPath := SrcPath(rootDir, spec)
	if exists, err := paths.DirExists(srcPath); err != nil {
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
	_ = gitcmd.RemoteSetURL(ctx, srcPath, "origin", remoteURL)
	return nil
}

func Exists(rootDir, repo string) (string, bool, error) {
	spec, err := repospec.Normalize(repo)
	if err != nil {
		return "", false, err
	}
	storePath := storePathForSpec(rootDir, spec)
	exists, err := paths.DirExists(storePath)
	if err != nil {
		return "", false, err
	}
	return storePath, exists, nil
}

func storePathForSpec(rootDir string, spec repospec.Spec) string {
	return StorePath(rootDir, spec)
}

func normalizeStore(ctx context.Context, storePath, display string, fetch bool) error {
	if _, err := gitcmd.Run(ctx, []string{"config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"}, gitcmd.Options{Dir: storePath}); err != nil {
		return err
	}
	defaultBranch, _ := localDefaultBranch(ctx, storePath)

	var remoteHash string
	remoteChecked := false
	grace := fetchGraceDuration()
	if !recentlyFetched(storePath, grace) || defaultBranch == "" {
		var err error
		defaultBranch, remoteHash, err = defaultBranchFromRemote(ctx, storePath)
		if err != nil {
			return err
		}
		remoteChecked = true
	}
	if defaultBranch != "" {
		_, _ = gitcmd.Run(ctx, []string{"symbolic-ref", "refs/remotes/origin/HEAD", fmt.Sprintf("refs/remotes/origin/%s", defaultBranch)}, gitcmd.Options{Dir: storePath})
	}

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

	needsFetch := remoteTrackingMissing
	if remoteChecked && defaultBranch != "" && remoteHash != "" && localHash != "" && localHash != remoteHash {
		needsFetch = true
	}
	if fetch && remoteChecked && (defaultBranch == "" || remoteHash == "") {
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
	hash, exists, err := gitcmd.ShowRef(ctx, storePath, ref)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	return hash, nil
}

func localHeadHash(ctx context.Context, storePath, branch string) (string, error) {
	ref := fmt.Sprintf("refs/heads/%s", branch)
	hash, exists, err := gitcmd.ShowRef(ctx, storePath, ref)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	return hash, nil
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
	out, err := gitcmd.WorktreeListPorcelain(ctx, storePath)
	if err != nil {
		return nil, err
	}
	branches := make(map[string]struct{})
	lines := strings.Split(strings.TrimSpace(out), "\n")
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

func fetchGraceDuration() time.Duration {
	val := strings.TrimSpace(os.Getenv("GWS_FETCH_GRACE_SECONDS"))
	if val == "" {
		return 30 * time.Second
	}
	secs, err := strconv.Atoi(val)
	if err != nil || secs < 0 {
		return 30 * time.Second
	}
	return time.Duration(secs) * time.Second
}

func recentlyFetched(storePath string, grace time.Duration) bool {
	if grace <= 0 {
		return false
	}
	info, err := os.Stat(filepath.Join(storePath, "FETCH_HEAD"))
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) <= grace
}

func localDefaultBranch(ctx context.Context, storePath string) (string, error) {
	ref, ok, err := gitcmd.SymbolicRef(ctx, storePath, "refs/remotes/origin/HEAD")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	if strings.HasPrefix(ref, "refs/remotes/origin/") {
		return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
	}
	return "", nil
}

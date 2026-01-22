package workspace_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
)

func TestRepoGetWorkspaceAddRemove(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gwst")
	t.Setenv("GIT_AUTHOR_EMAIL", "gwst@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gwst")
	t.Setenv("GIT_COMMITTER_EMAIL", "gwst@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gwst")

	remoteBase := filepath.Join(tmp, "remotes")
	remotePath := filepath.Join(remoteBase, "org", "repo.git")
	if err := os.MkdirAll(filepath.Dir(remotePath), 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, "", "init", "--bare", remotePath)

	seedDir := filepath.Join(tmp, "seed")
	runGit(t, "", "init", seedDir)
	runGit(t, seedDir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "init")
	runGit(t, seedDir, "remote", "add", "origin", remotePath)
	runGit(t, seedDir, "push", "origin", "main")
	runGit(t, "", "--git-dir", remotePath, "symbolic-ref", "HEAD", "refs/heads/main")

	configPath := filepath.Join(tmp, "gitconfig")
	fileURL := "file://" + filepath.ToSlash(remoteBase) + "/"
	configData := fmt.Sprintf("[url \"%s\"]\n\tinsteadOf = https://example.com/\n", fileURL)
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	repoSpec := "https://example.com/org/repo.git"
	store, err := repo.Get(ctx, rootDir, repoSpec)
	if err != nil {
		t.Fatalf("repo get: %v", err)
	}
	if _, err := os.Stat(store.StorePath); err != nil {
		t.Fatalf("store path missing: %v", err)
	}
	if _, err := workspace.New(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace new: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}
	worktreePath := workspace.WorktreePath(rootDir, "WS-1", "repo")
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("worktree missing: %v", err)
	}

	if err := workspace.Remove(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace remove: %v", err)
	}
	if _, err := os.Stat(workspace.WorkspaceDir(rootDir, "WS-1")); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists: %v", err)
	}
}

func TestWorkspaceAddSkipsFetchWhenUpToDate(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gwst")
	t.Setenv("GIT_AUTHOR_EMAIL", "gwst@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gwst")
	t.Setenv("GIT_COMMITTER_EMAIL", "gwst@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gwst")

	remoteBase := filepath.Join(tmp, "remotes")
	remotePath := filepath.Join(remoteBase, "org", "repo.git")
	if err := os.MkdirAll(filepath.Dir(remotePath), 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, "", "init", "--bare", remotePath)

	seedDir := filepath.Join(tmp, "seed")
	runGit(t, "", "init", seedDir)
	runGit(t, seedDir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "init")
	runGit(t, seedDir, "remote", "add", "origin", remotePath)
	runGit(t, seedDir, "push", "origin", "main")
	runGit(t, "", "--git-dir", remotePath, "symbolic-ref", "HEAD", "refs/heads/main")

	configPath := filepath.Join(tmp, "gitconfig")
	fileURL := "file://" + filepath.ToSlash(remoteBase) + "/"
	configData := fmt.Sprintf("[url \"%s\"]\n\tinsteadOf = https://example.com/\n", fileURL)
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	repoSpec := "https://example.com/org/repo.git"
	store, err := repo.Get(ctx, rootDir, repoSpec)
	if err != nil {
		t.Fatalf("repo get: %v", err)
	}

	fetchHeadPath := filepath.Join(store.StorePath, "FETCH_HEAD")
	_ = os.Remove(fetchHeadPath)

	if _, err := workspace.New(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace new: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}

	if _, err := os.Stat(fetchHeadPath); err != nil {
		t.Fatalf("expected FETCH_HEAD to be touched when store is up-to-date: %v", err)
	}
}

func TestWorkspaceAddFetchesEvenWithinGraceWhenFetchTrue(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gwst")
	t.Setenv("GIT_AUTHOR_EMAIL", "gwst@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gwst")
	t.Setenv("GIT_COMMITTER_EMAIL", "gwst@example.com")
	t.Setenv("GWST_FETCH_GRACE_SECONDS", "3600")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gwst")

	remoteBase := filepath.Join(tmp, "remotes")
	remotePath := filepath.Join(remoteBase, "org", "repo.git")
	if err := os.MkdirAll(filepath.Dir(remotePath), 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, "", "init", "--bare", remotePath)

	seedDir := filepath.Join(tmp, "seed")
	runGit(t, "", "init", seedDir)
	runGit(t, seedDir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "init")
	runGit(t, seedDir, "remote", "add", "origin", remotePath)
	runGit(t, seedDir, "push", "origin", "main")
	runGit(t, "", "--git-dir", remotePath, "symbolic-ref", "HEAD", "refs/heads/main")

	initialHash := revParse(t, seedDir, "HEAD")

	configPath := filepath.Join(tmp, "gitconfig")
	fileURL := "file://" + filepath.ToSlash(remoteBase) + "/"
	configData := fmt.Sprintf("[url \"%s\"]\n\tinsteadOf = https://example.com/\n", fileURL)
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	repoSpec := "https://example.com/org/repo.git"
	if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
		t.Fatalf("repo get: %v", err)
	}

	// remote advances after the last fetch; even within grace, fetch=true should observe the remote.
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello v2\n"), 0o644); err != nil {
		t.Fatalf("write seed file v2: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "second")
	runGit(t, seedDir, "push", "origin", "main")
	secondHash := revParse(t, seedDir, "HEAD")

	if _, err := workspace.New(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace new: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}

	worktreePath := workspace.WorktreePath(rootDir, "WS-1", "repo")
	head := revParse(t, worktreePath, "HEAD")
	if head != secondHash {
		t.Fatalf("expected worktree HEAD to reflect remote even within fetch grace; got %s, want %s (initial=%s)", head, secondHash, initialHash)
	}
}

func TestWorkspaceAddTracksRemoteBranchWhenPresent(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gwst")
	t.Setenv("GIT_AUTHOR_EMAIL", "gwst@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gwst")
	t.Setenv("GIT_COMMITTER_EMAIL", "gwst@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gwst")

	remoteBase := filepath.Join(tmp, "remotes")
	remotePath := filepath.Join(remoteBase, "org", "repo.git")
	if err := os.MkdirAll(filepath.Dir(remotePath), 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, "", "init", "--bare", remotePath)

	seedDir := filepath.Join(tmp, "seed")
	runGit(t, "", "init", seedDir)
	runGit(t, seedDir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "init")
	runGit(t, seedDir, "remote", "add", "origin", remotePath)
	runGit(t, seedDir, "push", "origin", "main")

	runGit(t, seedDir, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(seedDir, "FEATURE.md"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("write feature file: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "feature")
	runGit(t, seedDir, "push", "origin", "feature")
	featureHash := revParse(t, seedDir, "HEAD")

	runGit(t, "", "--git-dir", remotePath, "symbolic-ref", "HEAD", "refs/heads/main")

	configPath := filepath.Join(tmp, "gitconfig")
	fileURL := "file://" + filepath.ToSlash(remoteBase) + "/"
	configData := fmt.Sprintf("[url \"%s\"]\n\tinsteadOf = https://example.com/\n", fileURL)
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	repoSpec := "https://example.com/org/repo.git"
	store, err := repo.Get(ctx, rootDir, repoSpec)
	if err != nil {
		t.Fatalf("repo get: %v", err)
	}
	// Ensure we have only a remote-tracking ref for "feature" (no local head),
	// so workspace.AddWithBranch must decide based on refs/remotes/origin/feature.
	runGit(t, "", "--git-dir", store.StorePath, "update-ref", "-d", "refs/heads/feature")

	if _, err := workspace.New(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace new: %v", err)
	}
	if _, err := workspace.AddWithBranch(ctx, rootDir, "WS-1", repoSpec, "", "feature", "origin/main", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}

	worktreePath := workspace.WorktreePath(rootDir, "WS-1", "repo")
	head := revParse(t, worktreePath, "HEAD")
	if head != featureHash {
		t.Fatalf("expected worktree HEAD to match remote feature branch; got %s, want %s", head, featureHash)
	}
}

func TestResolveBaseRefFallsBackWhenRemoteHeadRefMissing(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gwst")
	t.Setenv("GIT_AUTHOR_EMAIL", "gwst@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gwst")
	t.Setenv("GIT_COMMITTER_EMAIL", "gwst@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gwst")

	remoteBase := filepath.Join(tmp, "remotes")
	remotePath := filepath.Join(remoteBase, "org", "repo.git")
	if err := os.MkdirAll(filepath.Dir(remotePath), 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, "", "init", "--bare", remotePath)

	seedDir := filepath.Join(tmp, "seed")
	runGit(t, "", "init", seedDir)
	runGit(t, seedDir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "init")
	runGit(t, seedDir, "remote", "add", "origin", remotePath)
	runGit(t, seedDir, "push", "origin", "main")
	runGit(t, "", "--git-dir", remotePath, "symbolic-ref", "HEAD", "refs/heads/main")

	configPath := filepath.Join(tmp, "gitconfig")
	fileURL := "file://" + filepath.ToSlash(remoteBase) + "/"
	configData := fmt.Sprintf("[url \"%s\"]\n\tinsteadOf = https://example.com/\n", fileURL)
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	repoSpec := "https://example.com/org/repo.git"
	store, err := repo.Get(ctx, rootDir, repoSpec)
	if err != nil {
		t.Fatalf("repo get: %v", err)
	}

	// Simulate the broken state:
	// - origin/HEAD points to origin/main
	// - but refs/remotes/origin/main is missing
	// - while refs/heads/main exists (common in bare clones)
	runGit(t, "", "--git-dir", store.StorePath, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	runGit(t, "", "--git-dir", store.StorePath, "update-ref", "-d", "refs/remotes/origin/main")

	baseRef, err := workspace.ResolveBaseRef(ctx, store.StorePath)
	if err != nil {
		t.Fatalf("resolve base ref: %v", err)
	}
	if baseRef != "refs/heads/main" {
		t.Fatalf("expected fallback to local head ref; got %q, want %q", baseRef, "refs/heads/main")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
}

func revParse(t *testing.T, dir string, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", ref)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse %s failed: %v", ref, err)
	}
	return strings.TrimSpace(string(out))
}

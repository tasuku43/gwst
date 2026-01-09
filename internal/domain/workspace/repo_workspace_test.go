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

	"github.com/tasuku43/gws/internal/domain/repo"
	"github.com/tasuku43/gws/internal/domain/workspace"
)

func TestRepoGetWorkspaceAddRemove(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gws")
	t.Setenv("GIT_AUTHOR_EMAIL", "gws@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gws")
	t.Setenv("GIT_COMMITTER_EMAIL", "gws@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gws")

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
	srcPath := filepath.Join(rootDir, "src", "example.com", "org", "repo")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("src path missing: %v", err)
	}

	if _, err := workspace.New(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace new: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}
	worktreePath := filepath.Join(rootDir, "workspaces", "WS-1", "repo")
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("worktree missing: %v", err)
	}

	if err := workspace.Remove(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace remove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootDir, "workspaces", "WS-1")); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists: %v", err)
	}
}

func TestWorkspaceAddSkipsFetchWhenUpToDate(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gws")
	t.Setenv("GIT_AUTHOR_EMAIL", "gws@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gws")
	t.Setenv("GIT_COMMITTER_EMAIL", "gws@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gws")

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

	if _, err := os.Stat(fetchHeadPath); err == nil {
		t.Fatalf("expected fetch to be skipped when store is up-to-date")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat FETCH_HEAD: %v", err)
	}
}

func TestWorkspaceAddRespectsFetchGrace(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gws")
	t.Setenv("GIT_AUTHOR_EMAIL", "gws@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gws")
	t.Setenv("GIT_COMMITTER_EMAIL", "gws@example.com")
	t.Setenv("GWS_FETCH_GRACE_SECONDS", "3600")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gws")

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

	// remote advances after the last fetch, but grace should skip re-fetch
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello v2\n"), 0o644); err != nil {
		t.Fatalf("write seed file v2: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "second")
	runGit(t, seedDir, "push", "origin", "main")

	if _, err := workspace.New(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace new: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}

	worktreePath := filepath.Join(rootDir, "workspaces", "WS-1", "repo")
	head := revParse(t, worktreePath, "HEAD")
	if head != initialHash {
		t.Fatalf("expected worktree HEAD to remain at initial hash due to fetch grace; got %s, want %s", head, initialHash)
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

package remove_repo_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/gion/internal/app/create"
	"github.com/tasuku43/gion/internal/app/remove_repo"
	"github.com/tasuku43/gion/internal/domain/repo"
	"github.com/tasuku43/gion/internal/domain/workspace"
)

func TestRemoveRepo_RejectsDirtyWhenNotAllowed(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gion")
	t.Setenv("GIT_AUTHOR_EMAIL", "gion@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gion")
	t.Setenv("GIT_COMMITTER_EMAIL", "gion@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gion")

	repoSpec := setupLocalRemoteRepo(t, tmp)
	if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
		t.Fatalf("repo get: %v", err)
	}

	if _, err := create.CreateWorkspace(ctx, rootDir, "WS-1", workspace.Metadata{Mode: workspace.MetadataModeRepo}); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}

	worktreePath := workspace.WorktreePath(rootDir, "WS-1", "repo")
	if err := os.WriteFile(filepath.Join(worktreePath, "DIRTY.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	err := remove_repo.RemoveRepo(ctx, rootDir, "WS-1", "repo", remove_repo.Options{
		AllowDirty:       false,
		AllowStatusError: false,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("expected dirty error, got: %v", err)
	}
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("worktree should remain: %v", err)
	}
}

func TestRemoveRepo_AllowsDirtyWhenAllowed(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gion")
	t.Setenv("GIT_AUTHOR_EMAIL", "gion@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gion")
	t.Setenv("GIT_COMMITTER_EMAIL", "gion@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gion")

	repoSpec := setupLocalRemoteRepo(t, tmp)
	if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
		t.Fatalf("repo get: %v", err)
	}

	if _, err := create.CreateWorkspace(ctx, rootDir, "WS-1", workspace.Metadata{Mode: workspace.MetadataModeRepo}); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}

	worktreePath := workspace.WorktreePath(rootDir, "WS-1", "repo")
	if err := os.WriteFile(filepath.Join(worktreePath, "DIRTY.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	if err := remove_repo.RemoveRepo(ctx, rootDir, "WS-1", "repo", remove_repo.Options{
		AllowDirty:       true,
		AllowStatusError: false,
	}); err != nil {
		t.Fatalf("remove repo: %v", err)
	}
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("worktree should be removed, stat err: %v", err)
	}
}

func setupLocalRemoteRepo(t *testing.T, tmp string) string {
	t.Helper()

	remoteBase := filepath.Join(tmp, "remotes")
	remotePath := filepath.Join(remoteBase, "example.com", "org", "repo.git")
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
	fileURL := "file://" + filepath.ToSlash(remoteBase) + "/example.com/"
	configData := fmt.Sprintf("[url \"%s\"]\n\tinsteadOf = https://example.com/\n", fileURL)
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	return "https://example.com/org/repo.git"
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

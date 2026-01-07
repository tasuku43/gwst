package workspace_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tasuku43/gws/internal/repo"
	"github.com/tasuku43/gws/internal/workspace"
)

func TestRepoGetWorkspaceAddRemove(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	remoteDir := filepath.Join(t.TempDir(), "remote")

	if err := os.MkdirAll(remoteDir, 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, remoteDir, "init", "-b", "main")
	runGit(t, remoteDir, "config", "user.email", "test@example.com")
	runGit(t, remoteDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, remoteDir, "add", "README.md")
	runGit(t, remoteDir, "commit", "-m", "init")

	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	configData := fmt.Sprintf("[url \"file://%s\"]\n\tinsteadOf = https://example.com/owner/repo\n", remoteDir)
	if err := os.WriteFile(gitConfigPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfigPath)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	repoSpec := "https://example.com/owner/repo"
	store, err := repo.Get(ctx, rootDir, repoSpec)
	if err != nil {
		t.Fatalf("repo.Get error: %v", err)
	}
	expectedStore := filepath.Join(rootDir, "bare", "example.com", "owner", "repo.git")
	if store.StorePath != expectedStore {
		t.Fatalf("expected store path %s, got %s", expectedStore, store.StorePath)
	}
	if _, err := os.Stat(store.StorePath); err != nil {
		t.Fatalf("store path missing: %v", err)
	}
	srcPath := filepath.Join(rootDir, "src", "example.com", "owner", "repo")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("src path missing: %v", err)
	}

	if _, err := workspace.New(ctx, rootDir, "TEST-1"); err != nil {
		t.Fatalf("workspace.New error: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "TEST-1", repoSpec, "", false); err != nil {
		t.Fatalf("workspace.Add error: %v", err)
	}

	worktreePath := filepath.Join(rootDir, "ws", "TEST-1", "repo")
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("worktree missing: %v", err)
	}

	if err := workspace.Remove(ctx, rootDir, "TEST-1"); err != nil {
		t.Fatalf("workspace.Remove error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootDir, "ws", "TEST-1")); !os.IsNotExist(err) {
		t.Fatalf("workspace dir still exists")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, string(out))
	}
}

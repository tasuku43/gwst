package cli

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

func TestRepoRemoveDeletesStoreWhenUnused(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gwst")

	repoSpec := "https://example.com/org/repo.git"
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		t.Fatalf("normalize repo spec: %v", err)
	}
	storePath := repo.StorePath(rootDir, spec)
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		t.Fatalf("mkdir store: %v", err)
	}

	if err := runRepoRemove(ctx, rootDir, []string{repoSpec}, true); err != nil {
		t.Fatalf("repo rm: %v", err)
	}
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Fatalf("store still exists: %v", err)
	}
}

func TestRepoRemoveFailsWhenWorkspaceUsesRepo(t *testing.T) {
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

	if _, err := workspace.New(ctx, rootDir, "WS-1"); err != nil {
		t.Fatalf("workspace new: %v", err)
	}
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}

	err = runRepoRemove(ctx, rootDir, []string{repoSpec}, true)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "WS-1") {
		t.Fatalf("expected workspace reference error, got: %v", err)
	}
	if _, err := os.Stat(store.StorePath); err != nil {
		t.Fatalf("store missing after failed remove: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
}

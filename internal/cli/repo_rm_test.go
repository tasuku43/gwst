package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/gion/internal/domain/repo"
	"github.com/tasuku43/gion/internal/domain/workspace"
)

func TestRepoRemoveDeletesStoreWhenUnused(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gion")

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
	t.Setenv("GIT_AUTHOR_NAME", "gion")
	t.Setenv("GIT_AUTHOR_EMAIL", "gion@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gion")
	t.Setenv("GIT_COMMITTER_EMAIL", "gion@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gion")

	repoSpec, _ := setupLocalRemoteRepoExampleDotCom(t, tmp)
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

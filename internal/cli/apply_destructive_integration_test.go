package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/gwst/internal/app/apply"
	"github.com/tasuku43/gwst/internal/app/create"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/ui"
)

func TestApply_NoPromptRejectsWorkspaceRemove(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gion")
	t.Setenv("GIT_AUTHOR_EMAIL", "gion@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gion")
	t.Setenv("GIT_COMMITTER_EMAIL", "gion@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gion")

	repoSpec, _ := setupLocalRemoteRepoExampleDotCom(t, tmp)
	if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
		t.Fatalf("repo get: %v", err)
	}

	if _, err := create.CreateWorkspace(ctx, rootDir, "WS-1", workspace.Metadata{Mode: workspace.MetadataModeRepo}); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := workspace.AddWithBranch(ctx, rootDir, "WS-1", repoSpec, "", "main", "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}
	worktreePath := workspace.WorktreePath(rootDir, "WS-1", "repo")
	runGit(t, worktreePath, "branch", "--set-upstream-to=origin/main")

	// Desired state is empty => destructive WorkspaceRemove.
	if err := manifest.Save(rootDir, manifest.File{Version: 1, Workspaces: map[string]manifest.Workspace{}}); err != nil {
		t.Fatalf("manifest save: %v", err)
	}
	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	var out bytes.Buffer
	renderer := ui.NewRenderer(&out, ui.DefaultTheme(), false)
	got, err := runApplyInternalWithPlan(ctx, rootDir, renderer, true, plan)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !got.HadChanges {
		t.Fatalf("expected HadChanges=true, got %+v", got)
	}
	if !strings.Contains(err.Error(), "destructive changes require confirmation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApply_ConfirmedRemovesDirtyWorkspace(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gion")
	t.Setenv("GIT_AUTHOR_EMAIL", "gion@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gion")
	t.Setenv("GIT_COMMITTER_EMAIL", "gion@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gion")

	repoSpec, _ := setupLocalRemoteRepoExampleDotCom(t, tmp)
	if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
		t.Fatalf("repo get: %v", err)
	}

	if _, err := create.CreateWorkspace(ctx, rootDir, "WS-1", workspace.Metadata{Mode: workspace.MetadataModeRepo}); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := workspace.AddWithBranch(ctx, rootDir, "WS-1", repoSpec, "", "main", "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}
	worktreePath := workspace.WorktreePath(rootDir, "WS-1", "repo")
	runGit(t, worktreePath, "branch", "--set-upstream-to=origin/main")

	// Make worktree dirty (untracked) to exercise the "AllowDirty" gate.
	if err := os.WriteFile(filepath.Join(worktreePath, "DIRTY.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	// Desired state is empty => WorkspaceRemove plan.
	if err := manifest.Save(rootDir, manifest.File{Version: 1, Workspaces: map[string]manifest.Workspace{}}); err != nil {
		t.Fatalf("manifest save: %v", err)
	}
	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if err := apply.Apply(ctx, rootDir, plan, apply.Options{
		AllowDirty:       true,
		AllowStatusError: true,
		PrefetchOK:       true,
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if _, err := os.Stat(workspace.WorkspaceDir(rootDir, "WS-1")); !os.IsNotExist(err) {
		t.Fatalf("workspace should be removed, stat err: %v", err)
	}
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("worktree should be removed, stat err: %v", err)
	}
}

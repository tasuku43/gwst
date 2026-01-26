package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/gion/internal/app/create"
	"github.com/tasuku43/gion/internal/app/manifestplan"
	"github.com/tasuku43/gion/internal/domain/manifest"
	"github.com/tasuku43/gion/internal/domain/repo"
	"github.com/tasuku43/gion/internal/domain/workspace"
	"github.com/tasuku43/gion/internal/ui"
)

func TestPlan_WorkspaceRemoveRisk_Unpushed(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(worktreePath, "LOCAL.txt"), []byte("local\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, worktreePath, "add", ".")
	runGit(t, worktreePath, "commit", "-m", "local commit")

	// Desired state is empty, so the existing workspace becomes a remove action.
	if err := manifest.Save(rootDir, manifest.File{Version: 1, Workspaces: map[string]manifest.Workspace{}}); err != nil {
		t.Fatalf("manifest save: %v", err)
	}

	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	var out bytes.Buffer
	renderer := ui.NewRenderer(&out, ui.DefaultTheme(), false)
	renderPlanChanges(ctx, rootDir, renderer, plan)

	got := out.String()
	if !strings.Contains(got, "- remove workspace WS-1") {
		t.Fatalf("expected workspace remove line, got:\n%s", got)
	}
	if !strings.Contains(got, "risk: unpushed") {
		t.Fatalf("expected unpushed risk line, got:\n%s", got)
	}
	if !strings.Contains(got, "ahead=1") {
		t.Fatalf("expected ahead=1, got:\n%s", got)
	}
}

func TestPlan_WorkspaceRemoveRisk_Diverged(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "gion")
	t.Setenv("GIT_AUTHOR_EMAIL", "gion@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gion")
	t.Setenv("GIT_COMMITTER_EMAIL", "gion@example.com")

	ctx := context.Background()
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "gion")

	repoSpec, remotePath := setupLocalRemoteRepoExampleDotCom(t, tmp)
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
	if err := os.WriteFile(filepath.Join(worktreePath, "LOCAL.txt"), []byte("local\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, worktreePath, "add", ".")
	runGit(t, worktreePath, "commit", "-m", "local commit")

	// Advance remote/main independently, then fetch in the worktree to show diverged.
	pusherDir := filepath.Join(tmp, "pusher")
	runGit(t, "", "clone", remotePath, pusherDir)
	runGit(t, pusherDir, "checkout", "main")
	if err := os.WriteFile(filepath.Join(pusherDir, "REMOTE.txt"), []byte("remote\n"), 0o644); err != nil {
		t.Fatalf("write remote file: %v", err)
	}
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "remote commit")
	runGit(t, pusherDir, "push", "origin", "main")
	runGit(t, worktreePath, "fetch", "origin", "main")

	if err := manifest.Save(rootDir, manifest.File{Version: 1, Workspaces: map[string]manifest.Workspace{}}); err != nil {
		t.Fatalf("manifest save: %v", err)
	}

	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	var out bytes.Buffer
	renderer := ui.NewRenderer(&out, ui.DefaultTheme(), false)
	renderPlanChanges(ctx, rootDir, renderer, plan)

	got := out.String()
	if !strings.Contains(got, "- remove workspace WS-1") {
		t.Fatalf("expected workspace remove line, got:\n%s", got)
	}
	if !strings.Contains(got, "risk: diverged") {
		t.Fatalf("expected diverged risk line, got:\n%s", got)
	}
	if !strings.Contains(got, "ahead=1") || !strings.Contains(got, "behind=1") {
		t.Fatalf("expected ahead=1 and behind=1, got:\n%s", got)
	}
}

func TestPlan_WorkspaceRemoveRisk_Dirty(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(worktreePath, "DIRTY.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := manifest.Save(rootDir, manifest.File{Version: 1, Workspaces: map[string]manifest.Workspace{}}); err != nil {
		t.Fatalf("manifest save: %v", err)
	}

	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	var out bytes.Buffer
	renderer := ui.NewRenderer(&out, ui.DefaultTheme(), false)
	renderPlanChanges(ctx, rootDir, renderer, plan)

	got := out.String()
	if !strings.Contains(got, "- remove workspace WS-1") {
		t.Fatalf("expected workspace remove line, got:\n%s", got)
	}
	if !strings.Contains(got, "risk: dirty") {
		t.Fatalf("expected dirty risk line, got:\n%s", got)
	}
	if !strings.Contains(got, "untracked=1") {
		t.Fatalf("expected dirty detail (untracked=1), got:\n%s", got)
	}
	if !strings.Contains(got, "DIRTY.txt") {
		t.Fatalf("expected dirty file to be listed, got:\n%s", got)
	}
}

func TestPlan_WorkspaceRemoveRisk_DirtyAndUnpushed(t *testing.T) {
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

	// Unpushed: local commit ahead of origin/main.
	if err := os.WriteFile(filepath.Join(worktreePath, "LOCAL.txt"), []byte("local\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, worktreePath, "add", ".")
	runGit(t, worktreePath, "commit", "-m", "local commit")

	// Dirty: untracked file after the commit.
	if err := os.WriteFile(filepath.Join(worktreePath, "DIRTY.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	if err := manifest.Save(rootDir, manifest.File{Version: 1, Workspaces: map[string]manifest.Workspace{}}); err != nil {
		t.Fatalf("manifest save: %v", err)
	}
	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	var out bytes.Buffer
	renderer := ui.NewRenderer(&out, ui.DefaultTheme(), false)
	renderPlanChanges(ctx, rootDir, renderer, plan)

	got := out.String()
	if !strings.Contains(got, "- remove workspace WS-1") {
		t.Fatalf("expected workspace remove line, got:\n%s", got)
	}
	// Risk line is "dirty" (more urgent than unpushed), but sync still shows ahead.
	if !strings.Contains(got, "risk: dirty") {
		t.Fatalf("expected dirty risk line, got:\n%s", got)
	}
	if !strings.Contains(got, "ahead=1") {
		t.Fatalf("expected ahead=1 in sync line, got:\n%s", got)
	}
	if !strings.Contains(got, "DIRTY.txt") {
		t.Fatalf("expected dirty file to be listed, got:\n%s", got)
	}
}

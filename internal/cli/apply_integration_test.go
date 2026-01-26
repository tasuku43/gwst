package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tasuku43/gion/internal/app/create"
	"github.com/tasuku43/gion/internal/app/manifestplan"
	"github.com/tasuku43/gion/internal/domain/manifest"
	"github.com/tasuku43/gion/internal/domain/repo"
	"github.com/tasuku43/gion/internal/domain/workspace"
	"github.com/tasuku43/gion/internal/infra/gitcmd"
	"github.com/tasuku43/gion/internal/ui"
)

func TestApply_BranchRenameInPlace_SucceedsWithDirtyWorktree(t *testing.T) {
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
	if _, err := workspace.Add(ctx, rootDir, "WS-1", repoSpec, "", true); err != nil {
		t.Fatalf("workspace add: %v", err)
	}
	worktreePath := workspace.WorktreePath(rootDir, "WS-1", "repo")
	// Ensure the worktree's origin URL stays in a normalized, repo_key-derivable form.
	// (Some git url rewrite configurations can make `git remote get-url` return file://...,
	// which breaks in-place branch-rename detection in the plan.)
	runGit(t, worktreePath, "remote", "set-url", "origin", repoSpec)

	// Make the worktree dirty to cover "complex git state" (uncommitted changes)
	// while ensuring in-place branch rename remains safe and works.
	if err := os.WriteFile(filepath.Join(worktreePath, "DIRTY.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	// Desired state: same repo, different branch -> in-place branch rename (non-destructive).
	desired := manifest.File{
		Version: 1,
		Workspaces: map[string]manifest.Workspace{
			"WS-1": {
				Mode: workspace.MetadataModeRepo,
				Repos: []manifest.Repo{
					{
						Alias:   "repo",
						RepoKey: "example.com/org/repo",
						Branch:  "WS-2",
					},
				},
			},
		},
	}
	if err := manifest.Save(rootDir, desired); err != nil {
		t.Fatalf("manifest save: %v", err)
	}

	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if planHasDestructiveChanges(plan) {
		t.Fatalf("expected non-destructive in-place branch rename plan, got changes: %+v", plan.Changes)
	}

	var buf bytes.Buffer
	renderer := ui.NewRenderer(&buf, ui.DefaultTheme(), false)
	got, err := runApplyInternalWithPlan(ctx, rootDir, renderer, true, plan)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !got.Applied {
		t.Fatalf("expected applied, got %+v", got)
	}

	branch, err := gitcmd.RevParse(ctx, worktreePath, "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	if branch != "WS-2" {
		t.Fatalf("branch: got %q, want %q", branch, "WS-2")
	}

	rewritten, err := manifest.Load(rootDir)
	if err != nil {
		t.Fatalf("manifest load: %v", err)
	}
	ws, ok := rewritten.Workspaces["WS-1"]
	if !ok || len(ws.Repos) != 1 {
		t.Fatalf("rewritten manifest missing workspace or repos: %+v", rewritten.Workspaces["WS-1"])
	}
	if ws.Repos[0].Branch != "WS-2" {
		t.Fatalf("rewritten branch: got %q, want %q", ws.Repos[0].Branch, "WS-2")
	}
}

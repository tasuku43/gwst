package ui

import "testing"

func TestWorkspaceRepoFilter_RepoMatchKeepsParent(t *testing.T) {
	workspaces := []WorkspaceChoice{
		{
			ID:          "alpha",
			Description: "first workspace",
			Repos: []PromptChoice{
				{
					Label:       "app (branch: feat)",
					Value:       "/ws/alpha/app",
					Description: "git@github.com:org/app.git",
					Details:     []string{"repo: github.com/org/app", "branch: feat"},
				},
			},
		},
		{
			ID:          "beta",
			Description: "second workspace",
			Repos: []PromptChoice{
				{
					Label:       "ops (branch: main)",
					Value:       "/ws/beta/ops",
					Description: "git@github.com:org/ops.git",
					Details:     []string{"repo: github.com/org/ops", "branch: main"},
				},
			},
		},
	}
	model := newWorkspaceRepoSelectModel("giongo", workspaces, DefaultTheme(), false)
	model.input.SetValue("github.com/org/app")
	filtered := model.filterWorkspaces()

	if len(filtered) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(filtered))
	}
	if filtered[0].ID != "alpha" {
		t.Fatalf("expected workspace alpha, got %s", filtered[0].ID)
	}
	if len(filtered[0].Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(filtered[0].Repos))
	}
}

func TestWorkspaceRepoFilter_WorkspaceMatchIncludesAllRepos(t *testing.T) {
	workspaces := []WorkspaceChoice{
		{
			ID:          "alpha",
			Description: "first workspace",
			Repos: []PromptChoice{
				{Label: "app (branch: feat)", Value: "/ws/alpha/app"},
				{Label: "api (branch: feat)", Value: "/ws/alpha/api"},
			},
		},
	}
	model := newWorkspaceRepoSelectModel("giongo", workspaces, DefaultTheme(), false)
	model.input.SetValue("alpha")
	filtered := model.filterWorkspaces()

	if len(filtered) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(filtered))
	}
	if len(filtered[0].Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(filtered[0].Repos))
	}
}

func TestWorkspaceRepoSelectionsPaths(t *testing.T) {
	workspaces := []WorkspaceChoice{
		{
			ID:            "alpha",
			WorkspacePath: "/ws/alpha",
			Repos: []PromptChoice{
				{Label: "app (branch: feat)", Value: "/ws/alpha/app"},
			},
		},
	}
	model := newWorkspaceRepoSelectModel("giongo", workspaces, DefaultTheme(), false)
	if len(model.selections) != 2 {
		t.Fatalf("expected 2 selections, got %d", len(model.selections))
	}
	if model.selections[0].Path != "/ws/alpha" {
		t.Fatalf("expected workspace path, got %s", model.selections[0].Path)
	}
	if model.selections[1].Path != "/ws/alpha/app" {
		t.Fatalf("expected repo path, got %s", model.selections[1].Path)
	}
}

func TestWorkspaceRepoFilter_FuzzyMatch(t *testing.T) {
	workspaces := []WorkspaceChoice{
		{
			ID:          "test",
			Description: "hogehoge",
			Repos: []PromptChoice{
				{
					Label:       "gion (branch: test)",
					Value:       "/ws/test/gion",
					Description: "github.com/tasuku43/gion",
				},
			},
		},
	}
	model := newWorkspaceRepoSelectModel("giongo", workspaces, DefaultTheme(), false)
	model.input.SetValue("testgion")
	filtered := model.filterWorkspaces()

	if len(filtered) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(filtered))
	}
	if len(filtered[0].Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(filtered[0].Repos))
	}
	if filtered[0].Repos[0].Label != "gion (branch: test)" {
		t.Fatalf("unexpected repo label: %s", filtered[0].Repos[0].Label)
	}
}

func TestWorkspaceRepoBestSelectionPrefersRepoMatch(t *testing.T) {
	workspaces := []WorkspaceChoice{
		{
			ID:          "consistent-branch-selection",
			Description: "desc",
			Repos: []PromptChoice{
				{
					Label:       "gion (branch: consistent-branch-selection)",
					Value:       "/ws/consistent-branch-selection/gion",
					Description: "github.com/tasuku43/gion",
				},
			},
		},
	}
	model := newWorkspaceRepoSelectModel("giongo", workspaces, DefaultTheme(), false)
	model.input.SetValue("consissgion")
	model.filtered = model.filterWorkspaces()
	model.rebuildSelections()

	best := model.bestSelectionIndex(model.input.Value())
	if best < 0 || best >= len(model.selections) {
		t.Fatalf("unexpected selection index: %d", best)
	}
	if model.selections[best].RepoIndex < 0 {
		t.Fatalf("expected repo selection, got workspace selection")
	}
}

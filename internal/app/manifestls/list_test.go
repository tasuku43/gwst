package manifestls

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tasuku43/gion/internal/domain/manifest"
)

func TestList_ClassifiesAppliedMissingDriftExtra(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(rootDir, "workspaces", "WS_APPLIED"), 0o755); err != nil {
		t.Fatalf("mkdir applied: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "workspaces", "WS_DRIFT"), 0o755); err != nil {
		t.Fatalf("mkdir drift: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "workspaces", "WS_EXTRA"), 0o755); err != nil {
		t.Fatalf("mkdir extra: %v", err)
	}

	file := manifest.File{
		Version: 1,
		Workspaces: map[string]manifest.Workspace{
			"WS_APPLIED": {Description: "applied", Repos: nil},
			"WS_MISSING": {Description: "missing", Repos: nil},
			"WS_DRIFT": {
				Description: "drift",
				Repos: []manifest.Repo{
					{Alias: "repo", RepoKey: "github.com/org/repo.git", Branch: "WS_DRIFT"},
				},
			},
		},
	}
	if err := manifest.Save(rootDir, file); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	result, err := List(ctx, rootDir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if result.Counts.Applied != 1 || result.Counts.Missing != 1 || result.Counts.Drift != 1 || result.Counts.Extra != 1 {
		t.Fatalf("unexpected counts: %+v", result.Counts)
	}

	got := map[string]DriftStatus{}
	for _, entry := range result.ManifestEntries {
		got[entry.WorkspaceID] = entry.Drift
	}
	if got["WS_APPLIED"] != DriftApplied {
		t.Fatalf("WS_APPLIED: expected %q, got %q", DriftApplied, got["WS_APPLIED"])
	}
	if got["WS_MISSING"] != DriftMissing {
		t.Fatalf("WS_MISSING: expected %q, got %q", DriftMissing, got["WS_MISSING"])
	}
	if got["WS_DRIFT"] != DriftDrift {
		t.Fatalf("WS_DRIFT: expected %q, got %q", DriftDrift, got["WS_DRIFT"])
	}

	if len(result.ExtraEntries) != 1 || result.ExtraEntries[0].WorkspaceID != "WS_EXTRA" || result.ExtraEntries[0].Drift != DriftExtra {
		t.Fatalf("unexpected extras: %+v", result.ExtraEntries)
	}
}

package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tasuku43/gws/internal/core/paths"
)

func TestCheckFindsIssues(t *testing.T) {
	rootDir := t.TempDir()
	now := time.Now().UTC()

	wsDir := filepath.Join(paths.WorkspacesRoot(rootDir), "WS1")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("mkdir ws: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(wsDir, "not-a-repo"), 0o755); err != nil {
		t.Fatalf("mkdir non-git dir: %v", err)
	}

	repoNoRemote := filepath.Join(paths.BareRoot(rootDir), "example.com", "org", "noremote.git")
	if err := os.MkdirAll(repoNoRemote, 0o755); err != nil {
		t.Fatalf("mkdir repo noremote: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoNoRemote, "config"), []byte("[core]\n\trepositoryformatversion = 0\n"), 0o644); err != nil {
		t.Fatalf("write config noremote: %v", err)
	}

	result, err := Check(context.Background(), rootDir, now)
	if err != nil {
		t.Fatalf("doctor check: %v", err)
	}
	kinds := map[string]int{}
	for _, issue := range result.Issues {
		kinds[issue.Kind]++
	}
	if kinds["missing_remote"] == 0 {
		t.Fatalf("expected missing_remote issue")
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warnings for non-git workspace entries")
	}
}

func TestCheckRootLayout(t *testing.T) {
	rootDir := t.TempDir()
	now := time.Now().UTC()

	result, err := Check(context.Background(), rootDir, now)
	if err != nil {
		t.Fatalf("doctor check: %v", err)
	}
	kinds := map[string]int{}
	for _, issue := range result.Issues {
		kinds[issue.Kind]++
	}
	if kinds["missing_root_dir"] == 0 {
		t.Fatalf("expected missing_root_dir issues")
	}
	if kinds["missing_root_file"] == 0 {
		t.Fatalf("expected missing_root_file issues")
	}
}

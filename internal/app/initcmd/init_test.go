package initcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCreatesAndSkips(t *testing.T) {
	rootDir := t.TempDir()

	result, err := Run(rootDir)
	if err != nil {
		t.Fatalf("init run: %v", err)
	}
	if len(result.CreatedDirs) == 0 {
		t.Fatalf("expected created dirs")
	}
	if len(result.CreatedFiles) == 0 {
		t.Fatalf("expected created files")
	}

	for _, dir := range []string{"bare", "workspaces"} {
		if _, err := os.Stat(filepath.Join(rootDir, dir)); err != nil {
			t.Fatalf("missing dir %s: %v", dir, err)
		}
	}
	configPath := filepath.Join(rootDir, "gwst.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("missing file gwst.yaml: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read gwst.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "presets:") {
		t.Fatalf("expected presets in gwst.yaml")
	}
	if !strings.Contains(content, "example:") {
		t.Fatalf("expected example preset in gwst.yaml")
	}
	if !strings.Contains(content, "git@github.com:octocat/Hello-World.git") {
		t.Fatalf("expected octocat/Hello-World repo in gwst.yaml")
	}
	if !strings.Contains(content, "git@github.com:octocat/Spoon-Knife.git") {
		t.Fatalf("expected octocat/Spoon-Knife repo in gwst.yaml")
	}

	second, err := Run(rootDir)
	if err != nil {
		t.Fatalf("second init run: %v", err)
	}
	if len(second.CreatedDirs) != 0 || len(second.CreatedFiles) != 0 {
		t.Fatalf("expected no created items on second run")
	}
	if len(second.SkippedDirs) == 0 || len(second.SkippedFiles) == 0 {
		t.Fatalf("expected skipped items on second run")
	}
}

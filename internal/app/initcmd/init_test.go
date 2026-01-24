package initcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/gwst/internal/domain/manifest"
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
	configPath := filepath.Join(rootDir, manifest.FileName)
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("missing file %s: %v", manifest.FileName, err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", manifest.FileName, err)
	}
	content := string(data)
	if !strings.Contains(content, "presets:") {
		t.Fatalf("expected presets in %s", manifest.FileName)
	}
	if !strings.Contains(content, "example:") {
		t.Fatalf("expected example preset in %s", manifest.FileName)
	}
	if !strings.Contains(content, "git@github.com:octocat/Hello-World.git") {
		t.Fatalf("expected octocat/Hello-World repo in %s", manifest.FileName)
	}
	if !strings.Contains(content, "git@github.com:octocat/Spoon-Knife.git") {
		t.Fatalf("expected octocat/Spoon-Knife repo in %s", manifest.FileName)
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

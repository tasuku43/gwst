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

	for _, dir := range []string{"bare", "src", "ws"} {
		if _, err := os.Stat(filepath.Join(rootDir, dir)); err != nil {
			t.Fatalf("missing dir %s: %v", dir, err)
		}
	}
	templatesPath := filepath.Join(rootDir, "templates.yaml")
	if _, err := os.Stat(templatesPath); err != nil {
		t.Fatalf("missing file templates.yaml: %v", err)
	}
	data, err := os.ReadFile(templatesPath)
	if err != nil {
		t.Fatalf("read templates.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "example:") {
		t.Fatalf("expected example template in templates.yaml")
	}
	if !strings.Contains(content, "git@github.com:github/docs.git") {
		t.Fatalf("expected github/docs repo in templates.yaml")
	}
	if !strings.Contains(content, "git@github.com:github/opensource.guide.git") {
		t.Fatalf("expected github/opensource.guide repo in templates.yaml")
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

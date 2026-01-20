package preset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tasuku43/gwst/internal/domain/manifest"
)

func TestValidatePresetsOK(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`version: 1
presets:
  app:
    repos:
      - git@github.com:org/app.git
  legacy:
    repos:
      - repo: git@github.com:org/legacy.git
workspaces: {}
`)
	path := filepath.Join(rootDir, manifest.FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write gwst.yaml: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(result.Issues))
	}
}

func TestValidatePresetsDuplicate(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`version: 1
presets:
  app:
    repos:
      - git@github.com:org/app.git
  app:
    repos:
      - git@github.com:org/other.git
workspaces: {}
`)
	path := filepath.Join(rootDir, manifest.FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write gwst.yaml: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindDuplicatePreset) {
		t.Fatalf("expected duplicate preset issue")
	}
}

func TestValidatePresetsMissingRepos(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`version: 1
presets:
  app:
    description: test
workspaces: {}
`)
	path := filepath.Join(rootDir, manifest.FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write gwst.yaml: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindMissingRequired) {
		t.Fatalf("expected missing required field issue")
	}
}

func TestValidatePresetsInvalidRepo(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`version: 1
presets:
  app:
    repos:
      - github.com/org/app
workspaces: {}
`)
	path := filepath.Join(rootDir, manifest.FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write gwst.yaml: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindInvalidRepoSpec) {
		t.Fatalf("expected invalid repo spec issue")
	}
}

func TestValidatePresetsInvalidYAML(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte("presets: [")
	path := filepath.Join(rootDir, manifest.FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write gwst.yaml: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindInvalidYAML) {
		t.Fatalf("expected invalid yaml issue")
	}
}

func TestValidatePresetsInvalidName(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`version: 1
presets:
  bad name:
    repos:
      - git@github.com:org/app.git
workspaces: {}
`)
	path := filepath.Join(rootDir, manifest.FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write gwst.yaml: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindInvalidPresetName) {
		t.Fatalf("expected invalid preset name issue")
	}
}

func hasIssueKind(issues []ValidationIssue, kind string) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}

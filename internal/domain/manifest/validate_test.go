package manifest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_MissingFileIsIssue(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()

	result, err := Validate(ctx, rootDir)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if !strings.Contains(result.Issues[0].Ref, "gwst.yaml") {
		t.Fatalf("expected gwst.yaml ref, got: %+v", result.Issues[0])
	}
}

func TestValidate_InvalidYAMLIsIssue(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "gwst.yaml"), []byte(":\n  -\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := Validate(ctx, rootDir)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Fatalf("expected issues")
	}
}

func TestValidate_ValidMinimalManifestOK(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	content := `
version: 1
workspaces:
  PROJ-1:
    mode: repo
    repos:
      - alias: api
        repo_key: github.com/org/api.git
        branch: PROJ-1
        base_ref: origin/main
`
	if err := os.WriteFile(filepath.Join(rootDir, "gwst.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := Validate(ctx, rootDir)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", result.Issues)
	}
}

func TestValidate_BadBranchAndDuplicateAlias(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	content := `
version: 1
workspaces:
  PROJ-2:
    repos:
      - alias: api
        repo_key: github.com/org/api.git
        branch: "bad branch"
      - alias: api
        repo_key: github.com/org/web.git
        branch: PROJ-2
`
	if err := os.WriteFile(filepath.Join(rootDir, "gwst.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := Validate(ctx, rootDir)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(result.Issues) < 2 {
		t.Fatalf("expected multiple issues, got: %+v", result.Issues)
	}
	var sawBranch bool
	var sawDup bool
	for _, issue := range result.Issues {
		if strings.Contains(issue.Ref, ".branch") {
			sawBranch = true
		}
		if strings.Contains(issue.Message, "duplicate alias") {
			sawDup = true
		}
	}
	if !sawBranch {
		t.Fatalf("expected branch issue, got: %+v", result.Issues)
	}
	if !sawDup {
		t.Fatalf("expected duplicate alias issue, got: %+v", result.Issues)
	}
}

func TestValidate_PresetModeRequiresPresetNameAndExistingPreset(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	content := `
version: 1
presets:
  webapp:
    repos:
      - git@github.com:org/api.git
workspaces:
  PROJ-3:
    mode: preset
    preset_name: missing
    repos: []
`
	if err := os.WriteFile(filepath.Join(rootDir, "gwst.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := Validate(ctx, rootDir)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	var sawMissingPreset bool
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "preset not found") {
			sawMissingPreset = true
		}
	}
	if !sawMissingPreset {
		t.Fatalf("expected missing preset issue, got: %+v", result.Issues)
	}
}

package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateTemplatesOK(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`templates:
  app:
    repos:
      - git@github.com:org/app.git
  legacy:
    repos:
      - repo: git@github.com:org/legacy.git
`)
	path := filepath.Join(rootDir, FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write templates: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(result.Issues))
	}
}

func TestValidateTemplatesDuplicate(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`templates:
  app:
    repos:
      - git@github.com:org/app.git
  app:
    repos:
      - git@github.com:org/other.git
`)
	path := filepath.Join(rootDir, FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write templates: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindDuplicateTemplate) {
		t.Fatalf("expected duplicate template issue")
	}
}

func TestValidateTemplatesMissingRepos(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`templates:
  app:
    description: test
`)
	path := filepath.Join(rootDir, FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write templates: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindMissingRequired) {
		t.Fatalf("expected missing required field issue")
	}
}

func TestValidateTemplatesInvalidRepo(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`templates:
  app:
    repos:
      - github.com/org/app
`)
	path := filepath.Join(rootDir, FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write templates: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindInvalidRepoSpec) {
		t.Fatalf("expected invalid repo spec issue")
	}
}

func TestValidateTemplatesInvalidYAML(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte("templates: [")
	path := filepath.Join(rootDir, FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write templates: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindInvalidYAML) {
		t.Fatalf("expected invalid yaml issue")
	}
}

func TestValidateTemplatesInvalidName(t *testing.T) {
	rootDir := t.TempDir()
	data := []byte(`templates:
  bad name:
    repos:
      - git@github.com:org/app.git
`)
	path := filepath.Join(rootDir, FileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write templates: %v", err)
	}
	result, err := Validate(rootDir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hasIssueKind(result.Issues, IssueKindInvalidTemplateName) {
		t.Fatalf("expected invalid template name issue")
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

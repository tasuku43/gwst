package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	rootDir := t.TempDir()
	if _, err := Load(rootDir); err == nil {
		t.Fatalf("expected error for missing templates file")
	}
}

func TestLoadAndNames(t *testing.T) {
	rootDir := t.TempDir()
	path := filepath.Join(rootDir, FileName)
	data := []byte(`
templates:
  app:
    repos:
      - git@github.com:org/app.git
  zzz:
    repos:
      - git@github.com:org/zzz.git
  legacy:
    repos:
      - repo: git@github.com:org/legacy.git
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write templates: %v", err)
	}
	file, err := Load(rootDir)
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}
	names := Names(file)
	want := []string{"app", "legacy", "zzz"}
	if len(names) != len(want) {
		t.Fatalf("expected %d names, got %d", len(want), len(names))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("name mismatch at %d: got %q want %q", i, names[i], want[i])
		}
	}
	legacy, ok := file.Templates["legacy"]
	if !ok || len(legacy.Repos) != 1 {
		t.Fatalf("legacy template not loaded")
	}
}

func TestValidateName(t *testing.T) {
	valid := []string{"a", "123", "hello-world", "a_b", strings.Repeat("a", 64)}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Fatalf("unexpected error for %q: %v", name, err)
		}
	}
	invalid := []string{"", " ", "a b", "*bad*", strings.Repeat("a", 65)}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Fatalf("expected error for %q", name)
		}
	}
}

func TestNormalizeRepos(t *testing.T) {
	repos := []string{" git@github.com:org/app.git ", "git@github.com:org/app.git", "", "https://github.com/org/other.git"}
	normalized := NormalizeRepos(repos)
	want := []string{"git@github.com:org/app.git", "https://github.com/org/other.git"}
	if len(normalized) != len(want) {
		t.Fatalf("unexpected length: got %d want %d", len(normalized), len(want))
	}
	for i := range want {
		if normalized[i] != want[i] {
			t.Fatalf("repo mismatch at %d: got %q want %q", i, normalized[i], want[i])
		}
	}
}

func TestSave(t *testing.T) {
	rootDir := t.TempDir()
	path := filepath.Join(rootDir, FileName)
	initial := []byte("templates:\n  old:\n    repos:\n      - git@github.com:org/old.git\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatalf("write templates: %v", err)
	}
	file, err := Load(rootDir)
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}
	file.Templates["new"] = Template{Repos: []string{"git@github.com:org/new.git"}}
	if err := Save(rootDir, file); err != nil {
		t.Fatalf("save templates: %v", err)
	}
	reloaded, err := Load(rootDir)
	if err != nil {
		t.Fatalf("reload templates: %v", err)
	}
	if _, ok := reloaded.Templates["new"]; !ok {
		t.Fatalf("new template not saved")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat templates: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("unexpected file mode: %v", info.Mode().Perm())
	}
}

func TestSaveMissingFile(t *testing.T) {
	rootDir := t.TempDir()
	file := File{Templates: map[string]Template{}}
	if err := Save(rootDir, file); err == nil {
		t.Fatalf("expected error when templates.yaml is missing")
	}
}

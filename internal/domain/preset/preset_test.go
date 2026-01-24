package preset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/gwst/internal/domain/manifest"
)

func TestLoadMissingFile(t *testing.T) {
	rootDir := t.TempDir()
	if _, err := Load(rootDir); err == nil {
		t.Fatalf("expected error for missing %s", manifest.FileName)
	}
}

func TestLoadAndNames(t *testing.T) {
	rootDir := t.TempDir()
	path := filepath.Join(rootDir, manifest.FileName)
	data := []byte(`
version: 1
presets:
  app:
    repos:
      - git@github.com:org/app.git
  zzz:
    repos:
      - git@github.com:org/zzz.git
  legacy:
    repos:
      - repo: git@github.com:org/legacy.git
workspaces: {}
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", manifest.FileName, err)
	}
	file, err := Load(rootDir)
	if err != nil {
		t.Fatalf("load %s: %v", manifest.FileName, err)
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
	legacy, ok := file.Presets["legacy"]
	if !ok || len(legacy.Repos) != 1 {
		t.Fatalf("legacy preset not loaded")
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
	path := filepath.Join(rootDir, manifest.FileName)
	initial := []byte("version: 1\npresets:\n  old:\n    repos:\n      - git@github.com:org/old.git\nworkspaces: {}\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatalf("write %s: %v", manifest.FileName, err)
	}
	file, err := Load(rootDir)
	if err != nil {
		t.Fatalf("load %s: %v", manifest.FileName, err)
	}
	file.Presets["new"] = Preset{Repos: []string{"git@github.com:org/new.git"}}
	if err := Save(rootDir, file); err != nil {
		t.Fatalf("save %s: %v", manifest.FileName, err)
	}
	reloaded, err := Load(rootDir)
	if err != nil {
		t.Fatalf("reload %s: %v", manifest.FileName, err)
	}
	if _, ok := reloaded.Presets["new"]; !ok {
		t.Fatalf("new preset not saved")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", manifest.FileName, err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("unexpected file mode: %v", info.Mode().Perm())
	}
}

func TestSaveMissingFile(t *testing.T) {
	rootDir := t.TempDir()
	file := File{Presets: map[string]Preset{}}
	if err := Save(rootDir, file); err != nil {
		t.Fatalf("unexpected error when %s is missing: %v", manifest.FileName, err)
	}
	if _, err := os.Stat(filepath.Join(rootDir, manifest.FileName)); err != nil {
		t.Fatalf("%s not created: %v", manifest.FileName, err)
	}
}

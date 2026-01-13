package repo

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/tasuku43/gws/internal/core/paths"
)

func TestListSrc(t *testing.T) {
	rootDir := t.TempDir()
	srcRoot := paths.SrcRoot(rootDir)

	repoA := filepath.Join(srcRoot, "example.com", "org", "alpha")
	repoB := filepath.Join(srcRoot, "example.com", "org", "beta")
	nonRepo := filepath.Join(srcRoot, "misc")

	for _, dir := range []string{
		filepath.Join(repoA, ".git"),
		filepath.Join(repoB, ".git"),
		nonRepo,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	entries, warnings, err := ListSrc(rootDir)
	if err != nil {
		t.Fatalf("ListSrc error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("ListSrc warnings: %v", warnings)
	}

	sort.Strings(entries)
	want := []string{repoA, repoB}
	if len(entries) != len(want) {
		t.Fatalf("ListSrc entries mismatch: got %v want %v", entries, want)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Fatalf("ListSrc entries mismatch: got %v want %v", entries, want)
		}
	}
}

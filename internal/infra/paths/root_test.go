package paths

import (
	"path/filepath"
	"testing"
)

func TestResolveRootFlagOverrides(t *testing.T) {
	t.Setenv("GION_ROOT", "/tmp/ignore")
	root, err := ResolveRoot("/tmp/custom")
	if err != nil {
		t.Fatalf("ResolveRoot error: %v", err)
	}
	if root != "/tmp/custom" {
		t.Fatalf("expected /tmp/custom, got %s", root)
	}
}

func TestResolveRootEnvOverridesConfig(t *testing.T) {
	t.Setenv("GION_ROOT", "/tmp/env-root")
	root, err := ResolveRoot("")
	if err != nil {
		t.Fatalf("ResolveRoot error: %v", err)
	}
	if root != "/tmp/env-root" {
		t.Fatalf("expected /tmp/env-root, got %s", root)
	}
}

func TestResolveRootDefault(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("HOME", temp)
	root, err := ResolveRoot("")
	if err != nil {
		t.Fatalf("ResolveRoot error: %v", err)
	}
	expected := filepath.Join(temp, "gion")
	if root != expected {
		t.Fatalf("expected %s, got %s", expected, root)
	}
}

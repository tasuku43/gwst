package cli

import "testing"

func TestShellCommandForOpenDefaults(t *testing.T) {
	path, args := shellCommandForOpen("")
	if path != "/bin/sh" {
		t.Fatalf("expected /bin/sh, got %q", path)
	}
	if len(args) != 1 || args[0] != "-i" {
		t.Fatalf("expected -i, got %v", args)
	}
}

func TestShellCommandForOpenInteractive(t *testing.T) {
	cases := []string{
		"/bin/bash",
		"/usr/bin/zsh",
		"/opt/homebrew/bin/fish",
		"/bin/sh",
		"/bin/ksh",
		"/bin/dash",
		"/bin/tcsh",
		"/bin/csh",
	}
	for _, path := range cases {
		_, args := shellCommandForOpen(path)
		if len(args) != 1 || args[0] != "-i" {
			t.Fatalf("expected -i for %q, got %v", path, args)
		}
	}
}

func TestShellCommandForOpenUnknown(t *testing.T) {
	path, args := shellCommandForOpen("/usr/bin/nu")
	if path != "/usr/bin/nu" {
		t.Fatalf("expected /usr/bin/nu, got %q", path)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}

func TestNestedOpenWorkspaceID(t *testing.T) {
	t.Setenv("GION_WORKSPACE", "")
	if got := nestedOpenWorkspaceID(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}

	t.Setenv("GION_WORKSPACE", "ISSUE-38")
	if got := nestedOpenWorkspaceID(); got != "ISSUE-38" {
		t.Fatalf("expected ISSUE-38, got %q", got)
	}
}

package cli

import (
	"os"
	"strings"
	"testing"
)

func TestRunGiongoRequiresTTY(t *testing.T) {
	originalArgs := os.Args
	originalIsTerminal := isTerminal
	defer func() {
		os.Args = originalArgs
		isTerminal = originalIsTerminal
	}()

	os.Args = []string{"giongo"}
	isTerminal = func(fd uintptr) bool { return false }

	err := RunGiongo()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "TTY") {
		t.Fatalf("expected TTY error, got %q", err.Error())
	}
}

func TestRunGiongoPrintUsesStderr(t *testing.T) {
	originalArgs := os.Args
	originalIsTerminal := isTerminal
	defer func() {
		os.Args = originalArgs
		isTerminal = originalIsTerminal
	}()

	os.Args = []string{"giongo", "--print"}
	isTerminal = func(fd uintptr) bool { return false }

	err := RunGiongo()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "TTY") {
		t.Fatalf("expected TTY error, got %q", err.Error())
	}
}

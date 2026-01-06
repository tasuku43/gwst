package gitcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/tasuku43/gws/internal/output"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Options struct {
	Dir string
	// ShowOutput prints stdout/stderr even when verbose is off.
	ShowOutput bool
}

func Run(ctx context.Context, args []string, opts Options) (Result, error) {
	if err := validateArgs(args); err != nil {
		return Result{
			Stderr:   err.Error(),
			ExitCode: -1,
		}, err
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if verbose {
		fmt.Fprintf(os.Stderr, "\x1b[36m%s$ git %s\x1b[0m\n", output.Indent, strings.Join(args, " "))
	}
	err := cmd.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(err),
	}
	if verbose || opts.ShowOutput {
		if result.Stdout != "" {
			writeIndented(os.Stderr, result.Stdout, output.Indent)
		}
		if result.Stderr != "" {
			writeIndented(os.Stderr, result.Stderr, output.Indent)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "%sexit: %d\n", output.Indent, result.ExitCode)
		}
	}
	if err != nil {
		return result, fmt.Errorf("git %v failed: %w", args, err)
	}
	return result, nil
}

func validateArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("git command is required")
	}
	if !isAllowedSubcommand(args[0]) {
		return fmt.Errorf("git subcommand %q is not allowed", args[0])
	}
	return nil
}

func isAllowedSubcommand(subcommand string) bool {
	_, ok := allowedSubcommands[subcommand]
	return ok
}

func writeIndented(w io.Writer, text, prefix string) {
	lines := strings.SplitAfter(text, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if line == "\n" {
			fmt.Fprint(w, prefix)
			continue
		}
		fmt.Fprint(w, prefix)
		fmt.Fprint(w, line)
	}
	if !strings.HasSuffix(text, "\n") {
		fmt.Fprintln(w)
	}
}

var allowedSubcommands = map[string]struct{}{
	"check-ref-format": {},
	"clone":            {},
	"config":           {},
	"fetch":            {},
	"init":             {},
	"ls-remote":        {},
	"remote":           {},
	"show-ref":         {},
	"symbolic-ref":     {},
	"status":           {},
	"update-ref":       {},
	"worktree":         {},
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return -1
	}
	return exitErr.ExitCode()
}

package gitcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/tasuku43/gwst/internal/infra/debuglog"
	"github.com/tasuku43/gwst/internal/infra/output"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Options struct {
	Dir string
	// ShowOutput prints stdout/stderr even when debug logging is off.
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

	trace := ""
	if debuglog.Enabled() {
		trace = debuglog.NewTrace("git")
		debuglog.LogCommand(trace, debuglog.FormatCommand("git", args))
	}
	err := cmd.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(err),
	}
	if debuglog.Enabled() {
		debuglog.LogStdoutLines(trace, result.Stdout)
		debuglog.LogStderrLines(trace, result.Stderr)
		debuglog.LogExit(trace, result.ExitCode)
	}
	if opts.ShowOutput {
		if result.Stdout != "" {
			output.LogLines(result.Stdout)
		}
		if result.Stderr != "" {
			output.LogLines(result.Stderr)
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

var allowedSubcommands = map[string]struct{}{
	"check-ref-format": {},
	"clone":            {},
	"config":           {},
	"fetch":            {},
	"init":             {},
	"ls-remote":        {},
	"rev-parse":        {},
	"remote":           {},
	"show-ref":         {},
	"symbolic-ref":     {},
	"status":           {},
	"update-ref":       {},
	"worktree":         {},
	"version":          {},
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

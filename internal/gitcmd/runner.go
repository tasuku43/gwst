package gitcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Options struct {
	Dir string
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

	err := cmd.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(err),
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
	"clone":    {},
	"fetch":    {},
	"init":     {},
	"status":   {},
	"worktree": {},
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

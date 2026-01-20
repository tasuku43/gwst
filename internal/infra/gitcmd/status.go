package gitcmd

import (
	"context"
	"fmt"
	"strings"
)

// StatusPorcelainV2 returns porcelain v2 status output with branch info.
func StatusPorcelainV2(ctx context.Context, dir string) (string, error) {
	res, err := Run(ctx, []string{"status", "--porcelain=v2", "-b"}, Options{Dir: dir})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return "", fmt.Errorf("git status failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return "", fmt.Errorf("git status failed: %w", err)
	}
	return res.Stdout, nil
}

// StatusShortBranch returns short status output with branch info.
func StatusShortBranch(ctx context.Context, dir string) (string, error) {
	res, err := Run(ctx, []string{"status", "--short", "--branch"}, Options{Dir: dir})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return "", fmt.Errorf("git status failed: %w: %s", err, strings.TrimSpace(res.Stderr))
		}
		return "", fmt.Errorf("git status failed: %w", err)
	}
	return res.Stdout, nil
}

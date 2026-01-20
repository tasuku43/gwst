package gitcmd

import (
	"context"
	"fmt"
	"strings"
)

// RemoteGetURL returns the remote URL for the given name.
func RemoteGetURL(ctx context.Context, dir, name string) (string, error) {
	res, err := Run(ctx, []string{"remote", "get-url", name}, Options{Dir: dir})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return "", fmt.Errorf("git remote get-url %s failed: %w: %s", name, err, strings.TrimSpace(res.Stderr))
		}
		return "", fmt.Errorf("git remote get-url %s failed: %w", name, err)
	}
	return strings.TrimSpace(res.Stdout), nil
}

// RemoteSetURL sets the remote URL for the given name.
func RemoteSetURL(ctx context.Context, dir, name, url string) error {
	res, err := Run(ctx, []string{"remote", "set-url", name, url}, Options{Dir: dir})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return fmt.Errorf("git remote set-url %s failed: %w: %s", name, err, strings.TrimSpace(res.Stderr))
		}
		return fmt.Errorf("git remote set-url %s failed: %w", name, err)
	}
	return nil
}

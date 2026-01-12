package gitcmd

import (
	"context"
	"fmt"
	"strings"
)

// RevParse runs git rev-parse and returns trimmed stdout.
func RevParse(ctx context.Context, dir string, args ...string) (string, error) {
	fullArgs := append([]string{"rev-parse"}, args...)
	res, err := Run(ctx, fullArgs, Options{Dir: dir})
	if err != nil {
		argLabel := strings.Join(args, " ")
		if strings.TrimSpace(res.Stderr) != "" {
			return "", fmt.Errorf("git rev-parse %s failed: %w: %s", argLabel, err, strings.TrimSpace(res.Stderr))
		}
		return "", fmt.Errorf("git rev-parse %s failed: %w", argLabel, err)
	}
	return strings.TrimSpace(res.Stdout), nil
}

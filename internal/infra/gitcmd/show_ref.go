package gitcmd

import (
	"context"
	"fmt"
	"strings"
)

// ShowRef verifies a ref and returns its hash when present.
func ShowRef(ctx context.Context, dir, ref string) (string, bool, error) {
	res, err := Run(ctx, []string{"show-ref", "--verify", ref}, Options{Dir: dir})
	if err == nil {
		fields := strings.Fields(strings.TrimSpace(res.Stdout))
		if len(fields) >= 1 {
			return fields[0], true, nil
		}
		return "", true, nil
	}
	if res.ExitCode == 1 || (res.ExitCode == 128 && strings.Contains(res.Stderr, "not a valid ref")) {
		return "", false, nil
	}
	if strings.TrimSpace(res.Stderr) != "" {
		return "", false, fmt.Errorf("git show-ref failed: %w: %s", err, strings.TrimSpace(res.Stderr))
	}
	return "", false, err
}

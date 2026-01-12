package gitcmd

import (
	"context"
	"fmt"
	"strings"
)

// SymbolicRef resolves a ref name. ok is false when the ref does not exist.
func SymbolicRef(ctx context.Context, dir, ref string) (string, bool, error) {
	res, err := Run(ctx, []string{"symbolic-ref", "--quiet", ref}, Options{Dir: dir})
	if err == nil {
		value := strings.TrimSpace(res.Stdout)
		if value == "" {
			return "", false, nil
		}
		return value, true, nil
	}
	if res.ExitCode == 1 {
		return "", false, nil
	}
	if strings.TrimSpace(res.Stderr) != "" {
		return "", false, fmt.Errorf("git symbolic-ref %s failed: %w: %s", ref, err, strings.TrimSpace(res.Stderr))
	}
	return "", false, err
}

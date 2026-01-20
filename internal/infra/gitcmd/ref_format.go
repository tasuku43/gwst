package gitcmd

import "context"

// CheckRefFormatBranch validates a branch/ref name using git check-ref-format.
func CheckRefFormatBranch(ctx context.Context, name string) error {
	_, err := Run(ctx, []string{"check-ref-format", "--branch", name}, Options{})
	return err
}

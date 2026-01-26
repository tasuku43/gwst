package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gion/internal/infra/gitcmd"
	"github.com/tasuku43/gion/internal/infra/paths"
)

func New(ctx context.Context, rootDir string, workspaceID string) (string, error) {
	if err := validateWorkspaceID(ctx, workspaceID); err != nil {
		return "", err
	}
	if rootDir == "" {
		return "", fmt.Errorf("root directory is required")
	}

	wsDir := WorkspaceDir(rootDir, workspaceID)
	if exists, err := paths.DirExists(wsDir); err != nil {
		return "", err
	} else if exists {
		return "", fmt.Errorf("workspace already exists: %s", wsDir)
	}

	if err := os.MkdirAll(wsDir, 0o750); err != nil {
		return "", fmt.Errorf("create workspace dir: %w", err)
	}

	return wsDir, nil
}

func validateWorkspaceID(ctx context.Context, workspaceID string) error {
	if workspaceID == "" {
		return fmt.Errorf("workspace id is required")
	}
	if strings.ContainsAny(workspaceID, `/\`) {
		return fmt.Errorf("invalid workspace id: must not contain path separators")
	}
	cleaned := filepath.Clean(workspaceID)
	if cleaned != workspaceID || cleaned == "." || cleaned == ".." {
		return fmt.Errorf("invalid workspace id: must not contain path traversal")
	}
	if err := gitcmd.CheckRefFormatBranch(ctx, workspaceID); err != nil {
		return fmt.Errorf("invalid workspace id: %w", err)
	}
	return nil
}

// ValidateWorkspaceID checks whether the given workspace id satisfies git's
// branch ref format rules. Workspace ids are used as branch names across
// worktrees.
func ValidateWorkspaceID(ctx context.Context, workspaceID string) error {
	return validateWorkspaceID(ctx, workspaceID)
}

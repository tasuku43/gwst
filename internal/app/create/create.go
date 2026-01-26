package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/tasuku43/gion/internal/domain/preset"
	"github.com/tasuku43/gion/internal/domain/workspace"
)

type PresetStepFunc func(repoSpec string, index, total int)

func CreateWorkspace(ctx context.Context, rootDir, workspaceID string, meta workspace.Metadata) (string, error) {
	wsDir, err := workspace.New(ctx, rootDir, workspaceID)
	if err != nil {
		return "", err
	}
	if err := workspace.SaveMetadata(wsDir, meta); err != nil {
		return "", err
	}
	return wsDir, nil
}

func ApplyPreset(ctx context.Context, rootDir, workspaceID string, preset preset.Preset, branches []string, step PresetStepFunc) error {
	total := len(preset.Repos)
	for i, repoSpec := range preset.Repos {
		branch := workspaceID
		if len(branches) == len(preset.Repos) && i < len(branches) && strings.TrimSpace(branches[i]) != "" {
			branch = branches[i]
		}
		if step != nil {
			step(repoSpec, i, total)
		}
		if _, err := workspace.AddWithBranch(ctx, rootDir, workspaceID, repoSpec, "", branch, "", false); err != nil {
			return err
		}
	}
	return nil
}

func FailWorkspaceMetadata(err error, rollbackErr error) error {
	if rollbackErr != nil {
		return fmt.Errorf("save workspace metadata failed: %w (rollback failed: %v)", err, rollbackErr)
	}
	return err
}

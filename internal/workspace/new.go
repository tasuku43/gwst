package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tasuku43/gws/internal/config"
	"github.com/tasuku43/gws/internal/gitcmd"
	"gopkg.in/yaml.v3"
)

const (
	manifestDirName  = ".gws"
	manifestFileName = "manifest.yaml"
)

func New(ctx context.Context, rootDir string, workspaceID string, cfg config.Config) (string, error) {
	if err := validateWorkspaceID(ctx, workspaceID); err != nil {
		return "", err
	}
	if rootDir == "" {
		return "", fmt.Errorf("root directory is required")
	}

	wsDir := filepath.Join(rootDir, "ws", workspaceID)
	if exists, err := pathExists(wsDir); err != nil {
		return "", err
	} else if exists {
		return "", fmt.Errorf("workspace already exists: %s", wsDir)
	}

	manifestDir := filepath.Join(wsDir, manifestDirName)
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		return "", fmt.Errorf("create workspace dir: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	manifest := Manifest{
		SchemaVersion: 1,
		WorkspaceID:   workspaceID,
		CreatedAt:     now,
		LastUsedAt:    now,
		Policy: Policy{
			Pinned:  false,
			TTLDays: cfg.Defaults.TTLDays,
		},
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("marshal manifest: %w", err)
	}
	manifestPath := filepath.Join(manifestDir, manifestFileName)
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write manifest: %w", err)
	}

	return wsDir, nil
}

func validateWorkspaceID(ctx context.Context, workspaceID string) error {
	if workspaceID == "" {
		return fmt.Errorf("workspace id is required")
	}
	_, err := gitcmd.Run(ctx, []string{"check-ref-format", "--branch", workspaceID}, gitcmd.Options{})
	if err != nil {
		return fmt.Errorf("invalid workspace id: %w", err)
	}
	return nil
}

func pathExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("path is not a directory: %s", path)
	}
	return true, nil
}

package initcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/infra/paths"
)

type Result struct {
	RootDir      string
	CreatedDirs  []string
	CreatedFiles []string
	SkippedFiles []string
	SkippedDirs  []string
}

func Run(rootDir string) (Result, error) {
	if rootDir == "" {
		return Result{}, fmt.Errorf("root directory is required")
	}

	result := Result{RootDir: rootDir}

	dirs := []string{
		paths.BareRoot(rootDir),
		paths.WorkspacesRoot(rootDir),
	}
	for _, dir := range dirs {
		if exists, err := paths.DirExists(dir); err != nil {
			return Result{}, err
		} else if exists {
			result.SkippedDirs = append(result.SkippedDirs, dir)
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Result{}, fmt.Errorf("create dir: %w", err)
		}
		result.CreatedDirs = append(result.CreatedDirs, dir)
	}

	configPath := filepath.Join(rootDir, manifest.FileName)
	if exists, err := paths.FileExists(configPath); err != nil {
		return Result{}, err
	} else if exists {
		result.SkippedFiles = append(result.SkippedFiles, configPath)
	} else {
		if err := writeManifest(configPath); err != nil {
			return Result{}, err
		}
		result.CreatedFiles = append(result.CreatedFiles, configPath)
	}

	return result, nil
}

func writeManifest(path string) error {
	file := manifest.File{
		Version:    1,
		Workspaces: map[string]manifest.Workspace{},
		Presets: map[string]manifest.Preset{
			"example": {
				Repos: []string{
					"git@github.com:octocat/Hello-World.git",
					"git@github.com:octocat/Spoon-Knife.git",
				},
			},
		},
	}
	data, err := manifest.Marshal(file)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", manifest.FileName, err)
	}
	return nil
}

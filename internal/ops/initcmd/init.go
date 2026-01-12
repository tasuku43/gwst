package initcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/tasuku43/gws/internal/core/paths"
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
		paths.SrcRoot(rootDir),
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

	templatesPath := filepath.Join(rootDir, "templates.yaml")
	if exists, err := paths.FileExists(templatesPath); err != nil {
		return Result{}, err
	} else if exists {
		result.SkippedFiles = append(result.SkippedFiles, templatesPath)
	} else {
		if err := writeTemplates(templatesPath); err != nil {
			return Result{}, err
		}
		result.CreatedFiles = append(result.CreatedFiles, templatesPath)
	}

	return result, nil
}

type templatesFile struct {
	Templates map[string]struct {
		Repos []string `yaml:"repos"`
	} `yaml:"templates"`
}

func writeTemplates(path string) error {
	file := templatesFile{
		Templates: map[string]struct {
			Repos []string `yaml:"repos"`
		}{
			"example": {
				Repos: []string{
					"git@github.com:octocat/Hello-World.git",
					"git@github.com:octocat/Spoon-Knife.git",
				},
			},
		},
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal templates: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write templates: %w", err)
	}
	return nil
}

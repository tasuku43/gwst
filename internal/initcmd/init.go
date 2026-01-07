package initcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
		filepath.Join(rootDir, "bare"),
		filepath.Join(rootDir, "src"),
		filepath.Join(rootDir, "ws"),
	}
	for _, dir := range dirs {
		if exists, err := dirExists(dir); err != nil {
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
	if exists, err := fileExists(templatesPath); err != nil {
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

func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
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
					"git@github.com:github/docs.git",
					"git@github.com:github/opensource.guide.git",
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

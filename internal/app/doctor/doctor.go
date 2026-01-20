package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/paths"
)

type Issue struct {
	Kind    string
	Path    string
	Message string
}

type Result struct {
	Issues   []Issue
	Warnings []error
}

type FixResult struct {
	Result
	Fixed []string
}

func Check(ctx context.Context, rootDir string, now time.Time) (Result, error) {
	if rootDir == "" {
		return Result{}, fmt.Errorf("root directory is required")
	}

	var issues []Issue
	issues = append(issues, checkRootLayout(rootDir)...)

	wsEntries, wsWarnings, err := workspace.List(rootDir)
	if err != nil {
		return Result{}, err
	}

	for _, entry := range wsEntries {
		repos, warnings, err := workspace.ScanRepos(ctx, entry.WorkspacePath)
		if err != nil {
			wsWarnings = append(wsWarnings, fmt.Errorf("workspace %s: %w", entry.WorkspaceID, err))
			continue
		}
		_ = repos
		for _, warning := range warnings {
			wsWarnings = append(wsWarnings, fmt.Errorf("workspace %s: %w", entry.WorkspaceID, warning))
		}
	}

	repoEntries, repoWarnings, err := repo.List(rootDir)
	if err != nil {
		return Result{}, err
	}

	for _, entry := range repoEntries {
		if ok, err := hasOriginRemote(entry.StorePath); err != nil {
			repoWarnings = append(repoWarnings, fmt.Errorf("repo %s: %w", entry.RepoKey, err))
		} else if !ok {
			issues = append(issues, Issue{
				Kind:    "missing_remote",
				Path:    entry.StorePath,
				Message: "origin remote not configured",
			})
		}
	}

	warnings := append(wsWarnings, repoWarnings...)
	return Result{Issues: issues, Warnings: warnings}, nil
}

func checkRootLayout(rootDir string) []Issue {
	var issues []Issue
	dirs := []struct {
		name string
		path string
	}{
		{name: "bare", path: paths.BareRoot(rootDir)},
		{name: "workspaces", path: paths.WorkspacesRoot(rootDir)},
	}
	for _, entry := range dirs {
		name := entry.name
		path := entry.path
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				issues = append(issues, Issue{
					Kind:    "missing_root_dir",
					Path:    path,
					Message: fmt.Sprintf("%s directory not found", name),
				})
				continue
			}
			issues = append(issues, Issue{
				Kind:    "invalid_root_dir",
				Path:    path,
				Message: fmt.Sprintf("cannot stat %s directory: %v", name, err),
			})
			continue
		}
		if !info.IsDir() {
			issues = append(issues, Issue{
				Kind:    "invalid_root_dir",
				Path:    path,
				Message: fmt.Sprintf("%s is not a directory", name),
			})
		}
	}

	files := []struct {
		name string
		path string
	}{
		{name: manifest.FileName, path: filepath.Join(rootDir, manifest.FileName)},
	}
	for _, entry := range files {
		name := entry.name
		path := entry.path
		exists, err := paths.FileExists(path)
		if err != nil {
			issues = append(issues, Issue{
				Kind:    "invalid_root_file",
				Path:    path,
				Message: fmt.Sprintf("cannot stat %s: %v", name, err),
			})
			continue
		}
		if !exists {
			issues = append(issues, Issue{
				Kind:    "missing_root_file",
				Path:    path,
				Message: fmt.Sprintf("%s not found", name),
			})
		}
	}
	return issues
}

func Fix(ctx context.Context, rootDir string, now time.Time) (FixResult, error) {
	result, err := Check(ctx, rootDir, now)
	if err != nil {
		return FixResult{}, err
	}

	var fixed []string

	return FixResult{Result: result, Fixed: fixed}, nil
}

func hasOriginRemote(storePath string) (bool, error) {
	configPath := filepath.Join(storePath, "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}
	content := string(data)
	if !strings.Contains(content, `remote "origin"`) {
		return false, nil
	}
	section := extractSection(content, `remote "origin"`)
	if section == "" {
		return false, nil
	}
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "url") && strings.Contains(line, "=") {
			return true, nil
		}
	}
	return false, nil
}

func extractSection(content, name string) string {
	var lines []string
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if strings.Contains(trimmed, name) {
				inSection = true
				continue
			}
			if inSection {
				break
			}
			continue
		}
		if inSection {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

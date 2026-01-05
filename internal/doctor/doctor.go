package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tasuku43/gws/internal/repo"
	"github.com/tasuku43/gws/internal/workspace"
)

const staleLockThreshold = 24 * time.Hour

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
	_ = ctx

	var issues []Issue

	wsEntries, wsWarnings, err := workspace.List(rootDir)
	if err != nil {
		return Result{}, err
	}

	for _, entry := range wsEntries {
		lockPath := filepath.Join(entry.WorkspacePath, ".gws", "lock")
		if stale, age, ok := staleLock(lockPath, now); ok && stale {
			issues = append(issues, Issue{
				Kind:    "stale_lock",
				Path:    lockPath,
				Message: fmt.Sprintf("stale workspace lock (age %s)", age),
			})
		}
		if entry.Manifest == nil {
			continue
		}
		for _, repoEntry := range entry.Manifest.Repos {
			if repoEntry.WorktreePath == "" {
				continue
			}
			if ok := dirExists(repoEntry.WorktreePath); !ok {
				issues = append(issues, Issue{
					Kind:    "missing_worktree",
					Path:    repoEntry.WorktreePath,
					Message: fmt.Sprintf("worktree missing for %s", repoEntry.Alias),
				})
			}
		}
	}

	repoEntries, repoWarnings, err := repo.List(rootDir)
	if err != nil {
		return Result{}, err
	}

	for _, entry := range repoEntries {
		lockPath := filepath.Join(entry.StorePath, ".gws", "lock")
		if stale, age, ok := staleLock(lockPath, now); ok && stale {
			issues = append(issues, Issue{
				Kind:    "stale_lock",
				Path:    lockPath,
				Message: fmt.Sprintf("stale repo lock (age %s)", age),
			})
		}
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

func Fix(ctx context.Context, rootDir string, now time.Time) (FixResult, error) {
	result, err := Check(ctx, rootDir, now)
	if err != nil {
		return FixResult{}, err
	}

	var fixed []string
	for _, issue := range result.Issues {
		if issue.Kind != "stale_lock" {
			continue
		}
		if err := os.Remove(issue.Path); err != nil && !os.IsNotExist(err) {
			return FixResult{}, fmt.Errorf("remove lock %s: %w", issue.Path, err)
		}
		fixed = append(fixed, issue.Path)
	}

	return FixResult{Result: result, Fixed: fixed}, nil
}

func staleLock(path string, now time.Time) (bool, time.Duration, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return false, 0, false
	}
	age := now.Sub(info.ModTime())
	return age > staleLockThreshold, age, true
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
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

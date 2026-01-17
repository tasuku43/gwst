package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gwst/internal/core/gitcmd"
	"github.com/tasuku43/gwst/internal/domain/repo"
)

func ScanRepos(ctx context.Context, wsDir string) ([]Repo, []error, error) {
	entries, err := os.ReadDir(wsDir)
	if err != nil {
		return nil, nil, err
	}
	var repos []Repo
	var warnings []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == ".gwst" {
			continue
		}
		repoPath := filepath.Join(wsDir, entry.Name())
		repo, warn, ok := inspectRepo(ctx, repoPath, entry.Name())
		if !ok {
			if warn != nil {
				warnings = append(warnings, warn)
			}
			continue
		}
		if warn != nil {
			warnings = append(warnings, warn)
		}
		repos = append(repos, repo)
	}
	return repos, warnings, nil
}

func inspectRepo(ctx context.Context, repoPath, alias string) (Repo, error, bool) {
	gitDir, err := gitRevParse(ctx, repoPath, "--git-dir")
	if err != nil {
		return Repo{}, fmt.Errorf("skip %s: not a git repo", repoPath), false
	}
	commonDir, err := gitRevParse(ctx, repoPath, "--git-common-dir")
	if err != nil {
		return Repo{}, fmt.Errorf("skip %s: %v", repoPath, err), false
	}

	absGitDir := resolveGitPath(repoPath, gitDir)
	absCommonDir := resolveGitPath(repoPath, commonDir)

	storePath := ""
	if absGitDir != absCommonDir {
		storePath = absCommonDir
	}

	branch := readBranch(ctx, repoPath)
	repoSpec, repoKey, warn := readRepoSpec(ctx, repoPath)

	repo := Repo{
		Alias:        alias,
		RepoSpec:     repoSpec,
		RepoKey:      repoKey,
		StorePath:    storePath,
		WorktreePath: repoPath,
		Branch:       branch,
	}
	return repo, warn, true
}

func gitRevParse(ctx context.Context, repoPath, arg string) (string, error) {
	return gitcmd.RevParse(ctx, repoPath, arg)
}

func resolveGitPath(repoPath, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(repoPath, value))
}

func readBranch(ctx context.Context, repoPath string) string {
	branch, ok, err := gitcmd.SymbolicRef(ctx, repoPath, "HEAD")
	if err == nil && ok && branch != "" {
		return strings.TrimPrefix(branch, "refs/heads/")
	}
	return ""
}

func readRepoSpec(ctx context.Context, repoPath string) (string, string, error) {
	remoteURL, err := gitcmd.RemoteGetURL(ctx, repoPath, "origin")
	if err != nil {
		return "", "", fmt.Errorf("origin remote missing: %w", err)
	}
	if remoteURL == "" {
		return "", "", fmt.Errorf("origin remote is empty")
	}
	spec, _, err := repo.Normalize(remoteURL)
	if err != nil {
		return remoteURL, "", fmt.Errorf("origin remote invalid: %s", err)
	}
	return remoteURL, spec.RepoKey, nil
}

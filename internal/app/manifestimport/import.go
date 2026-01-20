package manifestimport

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/paths"
)

type Result struct {
	Path     string
	Manifest manifest.File
	Warnings []error
}

func Import(ctx context.Context, rootDir string) (Result, error) {
	file, warnings, err := Build(ctx, rootDir)
	if err != nil {
		return Result{}, err
	}
	return Write(rootDir, file, warnings)
}

func Build(ctx context.Context, rootDir string) (manifest.File, []error, error) {
	if strings.TrimSpace(rootDir) == "" {
		return manifest.File{}, nil, fmt.Errorf("root directory is required")
	}
	wsRoot := paths.WorkspacesRoot(rootDir)
	exists, err := paths.DirExists(wsRoot)
	if err != nil {
		return manifest.File{}, nil, err
	}
	if !exists {
		return manifest.File{Version: 1, Workspaces: map[string]manifest.Workspace{}}, nil, nil
	}
	entries, err := os.ReadDir(wsRoot)
	if err != nil {
		return manifest.File{}, nil, err
	}

	file := manifest.File{
		Version:    1,
		Workspaces: map[string]manifest.Workspace{},
	}
	if existing, err := manifest.Load(rootDir); err == nil {
		file.Presets = existing.Presets
	}
	var warnings []error

	var workspaceIDs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		workspaceIDs = append(workspaceIDs, entry.Name())
	}
	sort.Strings(workspaceIDs)

	for _, wsID := range workspaceIDs {
		wsDir := workspace.WorkspaceDir(rootDir, wsID)
		meta, err := workspace.LoadMetadata(wsDir)
		if err != nil {
			warnings = append(warnings, fmt.Errorf("workspace %s metadata: %w", wsID, err))
		}

		repos, repoWarnings, err := workspace.ScanRepos(ctx, wsDir)
		if err != nil {
			warnings = append(warnings, fmt.Errorf("workspace %s repos: %w", wsID, err))
			continue
		}
		if len(repoWarnings) > 0 {
			for _, warn := range repoWarnings {
				warnings = append(warnings, fmt.Errorf("workspace %s repo: %w", wsID, warn))
			}
		}

		repoEntries := make([]manifest.Repo, 0, len(repos))
		for _, repoEntry := range repos {
			repoEntries = append(repoEntries, manifest.Repo{
				Alias:   strings.TrimSpace(repoEntry.Alias),
				RepoKey: strings.TrimSpace(repoEntry.RepoKey),
				Branch:  strings.TrimSpace(repoEntry.Branch),
			})
		}
		sort.Slice(repoEntries, func(i, j int) bool {
			return repoEntries[i].Alias < repoEntries[j].Alias
		})

		mode := strings.TrimSpace(meta.Mode)
		if mode == "" {
			mode = workspace.MetadataModeRepo
			warnings = append(warnings, fmt.Errorf("workspace %s metadata missing mode; defaulting to %s", wsID, mode))
		}

		wsEntry := manifest.Workspace{
			Description: strings.TrimSpace(meta.Description),
			Mode:        mode,
			PresetName:  strings.TrimSpace(meta.PresetName),
			SourceURL:   strings.TrimSpace(meta.SourceURL),
			Repos:       repoEntries,
		}
		file.Workspaces[wsID] = wsEntry
	}

	return file, warnings, nil
}

func Write(rootDir string, file manifest.File, warnings []error) (Result, error) {
	if err := manifest.Save(rootDir, file); err != nil {
		return Result{}, err
	}
	return Result{
		Path:     manifest.Path(rootDir),
		Manifest: file,
		Warnings: warnings,
	}, nil
}

func Path(rootDir string) string {
	return manifest.Path(rootDir)
}

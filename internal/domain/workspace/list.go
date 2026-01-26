package workspace

import (
	"fmt"
	"os"
	"strings"

	"github.com/tasuku43/gion/internal/infra/paths"
)

type Entry struct {
	WorkspaceID   string
	WorkspacePath string
	Description   string
}

func List(rootDir string) ([]Entry, []error, error) {
	wsRoot := WorkspacesRoot(rootDir)
	exists, err := paths.DirExists(wsRoot)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, nil
	}

	entries, err := os.ReadDir(wsRoot)
	if err != nil {
		return nil, nil, err
	}

	var results []Entry
	var warnings []error

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wsID := entry.Name()
		wsPath := WorkspaceDir(rootDir, wsID)

		description := ""
		meta, err := LoadMetadata(wsPath)
		if err != nil {
			warnings = append(warnings, fmt.Errorf("workspace %s metadata: %w", wsID, err))
		} else if strings.TrimSpace(meta.Description) != "" {
			description = strings.TrimSpace(meta.Description)
		}

		result := Entry{
			WorkspaceID:   wsID,
			WorkspacePath: wsPath,
			Description:   description,
		}
		results = append(results, result)
	}

	return results, warnings, nil
}

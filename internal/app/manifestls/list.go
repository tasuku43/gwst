package manifestls

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/tasuku43/gion/internal/app/manifestplan"
	"github.com/tasuku43/gion/internal/domain/workspace"
)

type DriftStatus string

const (
	DriftApplied DriftStatus = "applied"
	DriftMissing DriftStatus = "missing"
	DriftDrift   DriftStatus = "drift"
	DriftExtra   DriftStatus = "extra"
)

type Entry struct {
	WorkspaceID  string
	Drift        DriftStatus
	Risk         workspace.WorkspaceStateKind
	Description  string
	HasWorkspace bool
}

type Counts struct {
	Applied int
	Drift   int
	Missing int
	Extra   int
}

type Result struct {
	ManifestEntries []Entry
	ExtraEntries    []Entry
	Counts          Counts
	Warnings        []error
}

func List(ctx context.Context, rootDir string) (Result, error) {
	plan, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		return Result{}, err
	}
	desired := plan.Desired

	fsWorkspaces, fsWarnings, err := workspace.List(rootDir)
	if err != nil {
		return Result{}, err
	}
	warnings := append([]error{}, fsWarnings...)

	fsSet := make(map[string]workspace.Entry, len(fsWorkspaces))
	for _, entry := range fsWorkspaces {
		fsSet[entry.WorkspaceID] = entry
	}

	statusByWorkspaceID := make(map[string]DriftStatus, len(desired.Workspaces))
	for _, change := range plan.Changes {
		switch change.Kind {
		case manifestplan.WorkspaceAdd:
			statusByWorkspaceID[change.WorkspaceID] = DriftMissing
		case manifestplan.WorkspaceUpdate:
			statusByWorkspaceID[change.WorkspaceID] = DriftDrift
		default:
			// WorkspaceRemove is handled via filesystem scan (extra entries).
		}
	}

	var ids []string
	for id := range desired.Workspaces {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var counts Counts
	entries := make([]Entry, 0, len(ids))
	for _, id := range ids {
		ws := desired.Workspaces[id]
		drift := statusByWorkspaceID[id]
		if drift == "" {
			drift = DriftApplied
		}

		hasWorkspace := false
		risk := workspace.WorkspaceStateClean
		if _, ok := fsSet[id]; ok {
			hasWorkspace = true
			state, warn := bestEffortWorkspaceRisk(ctx, rootDir, id)
			if warn != nil {
				warnings = append(warnings, warn)
			}
			risk = state
		}

		entries = append(entries, Entry{
			WorkspaceID:  id,
			Drift:        drift,
			Risk:         risk,
			Description:  strings.TrimSpace(ws.Description),
			HasWorkspace: hasWorkspace,
		})

		switch drift {
		case DriftApplied:
			counts.Applied++
		case DriftMissing:
			counts.Missing++
		case DriftDrift:
			counts.Drift++
		}
	}

	var extraIDs []string
	for id := range fsSet {
		if _, ok := desired.Workspaces[id]; ok {
			continue
		}
		extraIDs = append(extraIDs, id)
	}
	sort.Strings(extraIDs)

	var extras []Entry
	for _, id := range extraIDs {
		risk, warn := bestEffortWorkspaceRisk(ctx, rootDir, id)
		if warn != nil {
			warnings = append(warnings, warn)
		}
		extras = append(extras, Entry{
			WorkspaceID:  id,
			Drift:        DriftExtra,
			Risk:         risk,
			HasWorkspace: true,
		})
		counts.Extra++
	}

	return Result{
		ManifestEntries: entries,
		ExtraEntries:    extras,
		Counts:          counts,
		Warnings:        warnings,
	}, nil
}

func bestEffortWorkspaceRisk(ctx context.Context, rootDir, workspaceID string) (workspace.WorkspaceStateKind, error) {
	state, err := workspace.State(ctx, rootDir, workspaceID)
	if err != nil {
		return workspace.WorkspaceStateUnknown, fmt.Errorf("workspace %s state: %w", workspaceID, err)
	}
	return aggregateRiskKind(state.Repos), nil
}

// aggregateRiskKind picks a single workspace risk label from repo risks.
//
// We keep "unknown" as a special-case top priority (can't confidently assert safety).
// When unknown is not present, we use a stable order: dirty > diverged > unpushed.
func aggregateRiskKind(repos []workspace.RepoState) workspace.WorkspaceStateKind {
	hasDirty := false
	hasUnknown := false
	hasDiverged := false
	hasUnpushed := false
	for _, repo := range repos {
		switch repo.Kind {
		case workspace.RepoStateUnknown:
			hasUnknown = true
		case workspace.RepoStateDirty:
			hasDirty = true
		case workspace.RepoStateDiverged:
			hasDiverged = true
		case workspace.RepoStateUnpushed:
			hasUnpushed = true
		}
	}
	switch {
	case hasUnknown:
		return workspace.WorkspaceStateUnknown
	case hasDirty:
		return workspace.WorkspaceStateDirty
	case hasDiverged:
		return workspace.WorkspaceStateDiverged
	case hasUnpushed:
		return workspace.WorkspaceStateUnpushed
	default:
		return workspace.WorkspaceStateClean
	}
}

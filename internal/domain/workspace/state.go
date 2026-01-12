package workspace

import (
	"context"
	"strings"
)

type WorkspaceStateKind string

const (
	WorkspaceStateClean    WorkspaceStateKind = "clean"
	WorkspaceStateDirty    WorkspaceStateKind = "dirty"
	WorkspaceStateUnpushed WorkspaceStateKind = "unpushed"
	WorkspaceStateDiverged WorkspaceStateKind = "diverged"
	WorkspaceStateUnknown  WorkspaceStateKind = "unknown"
)

type RepoStateKind string

const (
	RepoStateClean    RepoStateKind = "clean"
	RepoStateDirty    RepoStateKind = "dirty"
	RepoStateUnpushed RepoStateKind = "unpushed"
	RepoStateDiverged RepoStateKind = "diverged"
	RepoStateUnknown  RepoStateKind = "unknown"
)

type WorkspaceState struct {
	WorkspaceID string
	Kind        WorkspaceStateKind
	Repos       []RepoState
	Warnings    []error
}

type RepoState struct {
	Alias          string
	WorktreePath   string
	Upstream       string
	AheadCount     int
	BehindCount    int
	StagedCount    int
	UnstagedCount  int
	UntrackedCount int
	UnmergedCount  int
	Kind           RepoStateKind
	Error          error
}

func State(ctx context.Context, rootDir, workspaceID string) (WorkspaceState, error) {
	status, err := Status(ctx, rootDir, workspaceID)
	if err != nil {
		return WorkspaceState{}, err
	}
	return StateFromStatus(status), nil
}

func StateFromStatus(status StatusResult) WorkspaceState {
	repos := make([]RepoState, 0, len(status.Repos))
	for _, repo := range status.Repos {
		repos = append(repos, repoStateFromStatus(repo))
	}
	return WorkspaceState{
		WorkspaceID: status.WorkspaceID,
		Kind:        aggregateWorkspaceState(repos),
		Repos:       repos,
		Warnings:    status.Warnings,
	}
}

func repoStateFromStatus(repo RepoStatus) RepoState {
	state := RepoState{
		Alias:          repo.Alias,
		WorktreePath:   repo.WorktreePath,
		Upstream:       repo.Upstream,
		AheadCount:     repo.AheadCount,
		BehindCount:    repo.BehindCount,
		StagedCount:    repo.StagedCount,
		UnstagedCount:  repo.UnstagedCount,
		UntrackedCount: repo.UntrackedCount,
		UnmergedCount:  repo.UnmergedCount,
		Error:          repo.Error,
	}
	if repo.Error != nil {
		state.Kind = RepoStateUnknown
		return state
	}
	if repo.Dirty {
		state.Kind = RepoStateDirty
		return state
	}
	if strings.TrimSpace(repo.Upstream) == "" {
		state.Kind = RepoStateDiverged
		return state
	}
	if repo.AheadCount > 0 && repo.BehindCount > 0 {
		state.Kind = RepoStateDiverged
		return state
	}
	if repo.AheadCount > 0 {
		state.Kind = RepoStateUnpushed
		return state
	}
	if repo.BehindCount > 0 {
		state.Kind = RepoStateDiverged
		return state
	}
	state.Kind = RepoStateClean
	return state
}

func aggregateWorkspaceState(repos []RepoState) WorkspaceStateKind {
	hasDirty := false
	hasUnknown := false
	hasDiverged := false
	hasUnpushed := false
	for _, repo := range repos {
		switch repo.Kind {
		case RepoStateDirty:
			hasDirty = true
		case RepoStateUnknown:
			hasUnknown = true
		case RepoStateDiverged:
			hasDiverged = true
		case RepoStateUnpushed:
			hasUnpushed = true
		}
	}
	switch {
	case hasDirty:
		return WorkspaceStateDirty
	case hasUnknown:
		return WorkspaceStateUnknown
	case hasDiverged:
		return WorkspaceStateDiverged
	case hasUnpushed:
		return WorkspaceStateUnpushed
	default:
		return WorkspaceStateClean
	}
}

func RequiresRemoveConfirmation(kind WorkspaceStateKind) bool {
	switch kind {
	case WorkspaceStateUnpushed, WorkspaceStateDiverged, WorkspaceStateUnknown:
		return true
	default:
		return false
	}
}

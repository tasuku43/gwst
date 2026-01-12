package workspace

import (
	"errors"
	"testing"
)

var errTest = errors.New("boom")

func TestStateFromStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   StatusResult
		want WorkspaceStateKind
	}{
		{
			name: "clean",
			in: StatusResult{
				WorkspaceID: "ws-clean",
				Repos: []RepoStatus{
					{Alias: "app", Upstream: "origin/main"},
				},
			},
			want: WorkspaceStateClean,
		},
		{
			name: "dirty",
			in: StatusResult{
				WorkspaceID: "ws-dirty",
				Repos: []RepoStatus{
					{Alias: "app", Dirty: true, StagedCount: 1},
				},
			},
			want: WorkspaceStateDirty,
		},
		{
			name: "unpushed",
			in: StatusResult{
				WorkspaceID: "ws-unpushed",
				Repos: []RepoStatus{
					{Alias: "app", Upstream: "origin/main", AheadCount: 2},
				},
			},
			want: WorkspaceStateUnpushed,
		},
		{
			name: "diverged",
			in: StatusResult{
				WorkspaceID: "ws-diverged",
				Repos: []RepoStatus{
					{Alias: "app", Upstream: "origin/main", BehindCount: 1},
				},
			},
			want: WorkspaceStateDiverged,
		},
		{
			name: "unknown",
			in: StatusResult{
				WorkspaceID: "ws-unknown",
				Repos: []RepoStatus{
					{Alias: "app", Error: errTest},
				},
			},
			want: WorkspaceStateUnknown,
		},
		{
			name: "dirty_overrides_unknown",
			in: StatusResult{
				WorkspaceID: "ws-mixed",
				Repos: []RepoStatus{
					{Alias: "app", Error: errTest},
					{Alias: "api", Dirty: true},
				},
			},
			want: WorkspaceStateDirty,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := StateFromStatus(tc.in)
			if got.Kind != tc.want {
				t.Fatalf("StateFromStatus() kind = %q, want %q", got.Kind, tc.want)
			}
		})
	}
}

func TestRequiresRemoveConfirmation(t *testing.T) {
	t.Parallel()

	if RequiresRemoveConfirmation(WorkspaceStateClean) {
		t.Fatalf("WorkspaceStateClean should not require confirmation")
	}
	if RequiresRemoveConfirmation(WorkspaceStateDirty) {
		t.Fatalf("WorkspaceStateDirty should not require confirmation")
	}
	for _, kind := range []WorkspaceStateKind{WorkspaceStateUnpushed, WorkspaceStateDiverged, WorkspaceStateUnknown} {
		if !RequiresRemoveConfirmation(kind) {
			t.Fatalf("%s should require confirmation", kind)
		}
	}
}

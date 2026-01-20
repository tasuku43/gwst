package ui

import "testing"

func TestFormatInputsHeader(t *testing.T) {
	cases := []struct {
		title       string
		preset      string
		workspaceID string
		want        string
	}{
		{
			title: "gwst create",
			want:  "gwst create",
		},
		{
			title:  "gwst create",
			preset: "app",
			want:   "gwst create (preset: app)",
		},
		{
			title:       "gwst create",
			workspaceID: "ABC-123",
			want:        "gwst create (workspace id: ABC-123)",
		},
		{
			title:       "gwst create",
			preset:      "app",
			workspaceID: "ABC-123",
			want:        "gwst create (preset: app, workspace id: ABC-123)",
		},
	}

	for _, tc := range cases {
		if got := formatInputsHeader(tc.title, tc.preset, tc.workspaceID); got != tc.want {
			t.Fatalf("formatInputsHeader(%q, %q, %q) = %q, want %q", tc.title, tc.preset, tc.workspaceID, got, tc.want)
		}
	}
}

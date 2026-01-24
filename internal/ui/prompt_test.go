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
			title: "gwiac manifest add",
			want:  "gwiac manifest add",
		},
		{
			title:  "gwiac manifest add",
			preset: "app",
			want:   "gwiac manifest add (preset: app)",
		},
		{
			title:       "gwiac manifest add",
			workspaceID: "ABC-123",
			want:        "gwiac manifest add (workspace id: ABC-123)",
		},
		{
			title:       "gwiac manifest add",
			preset:      "app",
			workspaceID: "ABC-123",
			want:        "gwiac manifest add (preset: app, workspace id: ABC-123)",
		},
	}

	for _, tc := range cases {
		if got := formatInputsHeader(tc.title, tc.preset, tc.workspaceID); got != tc.want {
			t.Fatalf("formatInputsHeader(%q, %q, %q) = %q, want %q", tc.title, tc.preset, tc.workspaceID, got, tc.want)
		}
	}
}

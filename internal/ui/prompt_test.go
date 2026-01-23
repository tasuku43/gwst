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
			title: "gwst manifest add",
			want:  "gwst manifest add",
		},
		{
			title:  "gwst manifest add",
			preset: "app",
			want:   "gwst manifest add (preset: app)",
		},
		{
			title:       "gwst manifest add",
			workspaceID: "ABC-123",
			want:        "gwst manifest add (workspace id: ABC-123)",
		},
		{
			title:       "gwst manifest add",
			preset:      "app",
			workspaceID: "ABC-123",
			want:        "gwst manifest add (preset: app, workspace id: ABC-123)",
		},
	}

	for _, tc := range cases {
		if got := formatInputsHeader(tc.title, tc.preset, tc.workspaceID); got != tc.want {
			t.Fatalf("formatInputsHeader(%q, %q, %q) = %q, want %q", tc.title, tc.preset, tc.workspaceID, got, tc.want)
		}
	}
}

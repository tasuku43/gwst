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
			title: "gion manifest add",
			want:  "gion manifest add",
		},
		{
			title:  "gion manifest add",
			preset: "app",
			want:   "gion manifest add (preset: app)",
		},
		{
			title:       "gion manifest add",
			workspaceID: "ABC-123",
			want:        "gion manifest add (workspace id: ABC-123)",
		},
		{
			title:       "gion manifest add",
			preset:      "app",
			workspaceID: "ABC-123",
			want:        "gion manifest add (preset: app, workspace id: ABC-123)",
		},
	}

	for _, tc := range cases {
		if got := formatInputsHeader(tc.title, tc.preset, tc.workspaceID); got != tc.want {
			t.Fatalf("formatInputsHeader(%q, %q, %q) = %q, want %q", tc.title, tc.preset, tc.workspaceID, got, tc.want)
		}
	}
}

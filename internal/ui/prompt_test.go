package ui

import "testing"

func TestFormatInputsHeader(t *testing.T) {
	cases := []struct {
		title       string
		template    string
		workspaceID string
		want        string
	}{
		{
			title: "gws new",
			want:  "gws new",
		},
		{
			title:    "gws new",
			template: "app",
			want:     "gws new (template: app)",
		},
		{
			title:       "gws new",
			workspaceID: "ABC-123",
			want:        "gws new (workspace id: ABC-123)",
		},
		{
			title:       "gws new",
			template:    "app",
			workspaceID: "ABC-123",
			want:        "gws new (template: app, workspace id: ABC-123)",
		},
	}

	for _, tc := range cases {
		if got := formatInputsHeader(tc.title, tc.template, tc.workspaceID); got != tc.want {
			t.Fatalf("formatInputsHeader(%q, %q, %q) = %q, want %q", tc.title, tc.template, tc.workspaceID, got, tc.want)
		}
	}
}

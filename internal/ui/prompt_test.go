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
			title: "gws create",
			want:  "gws create",
		},
		{
			title:    "gws create",
			template: "app",
			want:     "gws create (template: app)",
		},
		{
			title:       "gws create",
			workspaceID: "ABC-123",
			want:        "gws create (workspace id: ABC-123)",
		},
		{
			title:       "gws create",
			template:    "app",
			workspaceID: "ABC-123",
			want:        "gws create (template: app, workspace id: ABC-123)",
		},
	}

	for _, tc := range cases {
		if got := formatInputsHeader(tc.title, tc.template, tc.workspaceID); got != tc.want {
			t.Fatalf("formatInputsHeader(%q, %q, %q) = %q, want %q", tc.title, tc.template, tc.workspaceID, got, tc.want)
		}
	}
}

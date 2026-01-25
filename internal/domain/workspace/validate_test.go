package workspace

import (
	"context"
	"strings"
	"testing"
)

func TestValidateWorkspaceID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name      string
		id        string
		wantError bool
		contains  string
	}{
		{name: "valid", id: "PROJ-123"},
		{name: "slash", id: "feat/one", wantError: true, contains: "path separators"},
		{name: "backslash", id: `feat\one`, wantError: true, contains: "path separators"},
		{name: "dot", id: ".", wantError: true, contains: "path traversal"},
		{name: "dotdot", id: "..", wantError: true, contains: "path traversal"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateWorkspaceID(ctx, tc.id)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error for %q", tc.id)
				}
				if tc.contains != "" && !strings.Contains(err.Error(), tc.contains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.contains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.id, err)
			}
		})
	}
}

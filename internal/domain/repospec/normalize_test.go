package repospec

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantKey string
		wantErr bool
	}{
		{
			name:    "ssh",
			input:   "git@github.com:org/repo.git",
			wantKey: "github.com/org/repo",
		},
		{
			name:    "https",
			input:   "https://github.com/org/repo",
			wantKey: "github.com/org/repo",
		},
		{
			name:    "file",
			input:   "file:///tmp/mirrors/example.com/org/repo.git",
			wantKey: "example.com/org/repo",
		},
		{
			name:    "shorthand",
			input:   "github.com/org/repo.git",
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   "org/repo",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := Normalize(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if spec.RepoKey != tc.wantKey {
				t.Fatalf("repo key mismatch: got %q want %q", spec.RepoKey, tc.wantKey)
			}
		})
	}
}

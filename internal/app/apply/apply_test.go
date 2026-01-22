package apply

import "testing"

func TestUpdateBaseBranchCandidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		candidate     string
		mixed         bool
		input         string
		wantCandidate string
		wantMixed     bool
	}{
		{
			name:          "empty input keeps empty",
			candidate:     "",
			mixed:         false,
			input:         "",
			wantCandidate: "",
			wantMixed:     false,
		},
		{
			name:          "first base branch sets candidate",
			candidate:     "",
			mixed:         false,
			input:         "origin/main",
			wantCandidate: "origin/main",
			wantMixed:     false,
		},
		{
			name:          "same base branch keeps candidate",
			candidate:     "origin/main",
			mixed:         false,
			input:         "origin/main",
			wantCandidate: "origin/main",
			wantMixed:     false,
		},
		{
			name:          "different base branch marks mixed",
			candidate:     "origin/main",
			mixed:         false,
			input:         "origin/master",
			wantCandidate: "origin/main",
			wantMixed:     true,
		},
		{
			name:          "once mixed stays mixed",
			candidate:     "origin/main",
			mixed:         true,
			input:         "origin/main",
			wantCandidate: "origin/main",
			wantMixed:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCandidate, gotMixed := updateBaseBranchCandidate(tt.candidate, tt.mixed, tt.input)
			if gotCandidate != tt.wantCandidate {
				t.Fatalf("candidate: got %q, want %q", gotCandidate, tt.wantCandidate)
			}
			if gotMixed != tt.wantMixed {
				t.Fatalf("mixed: got %v, want %v", gotMixed, tt.wantMixed)
			}
		})
	}
}

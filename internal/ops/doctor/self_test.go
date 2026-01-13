package doctor

import "testing"

func TestParseGitVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		want   gitVersion
		wantOK bool
	}{
		{name: "standard", input: "git version 2.39.1", want: gitVersion{2, 39, 1}, wantOK: true},
		{name: "apple", input: "git version 2.39.1 (Apple Git-143)", want: gitVersion{2, 39, 1}, wantOK: true},
		{name: "windows", input: "git version 2.42.0.windows.1", want: gitVersion{2, 42, 0}, wantOK: true},
		{name: "no patch", input: "git version 2.30", want: gitVersion{2, 30, 0}, wantOK: true},
		{name: "invalid", input: "version unknown", wantOK: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseGitVersion(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ok mismatch: got %v want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got != tt.want {
				t.Fatalf("version mismatch: got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestGitVersionLess(t *testing.T) {
	t.Parallel()
	if !(gitVersion{major: 2, minor: 19, patch: 0}.Less(minGitVersion)) {
		t.Fatalf("expected version to be less than minimum")
	}
	if (gitVersion{major: 2, minor: 20, patch: 0}.Less(minGitVersion)) {
		t.Fatalf("expected version to meet minimum")
	}
	if (gitVersion{major: 3, minor: 0, patch: 0}.Less(minGitVersion)) {
		t.Fatalf("expected version to exceed minimum")
	}
}

func TestOsCaveats(t *testing.T) {
	t.Parallel()
	if len(osCaveats("windows")) == 0 {
		t.Fatalf("expected windows caveat")
	}
	if len(osCaveats("darwin")) != 0 {
		t.Fatalf("expected no caveat for darwin")
	}
}

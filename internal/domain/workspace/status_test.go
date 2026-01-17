package workspace

import "testing"

func TestParseStatusPorcelainV2Counts(t *testing.T) {
	out := "# branch.oid 94a67ef\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +2 -1\n1 .M N... 100644 100644 100644 abcdef0 abcdef0 file.txt\n? new.txt\nu UU N... 100644 100644 100644 abcdef0 abcdef0 abcdef0 conflict.txt\n"
	branch, upstream, head, detached, headMissing, dirty, untracked, staged, unstaged, unmerged, ahead, behind := parseStatusPorcelainV2(out, "fallback")

	if branch != "main" {
		t.Fatalf("branch = %q, want main", branch)
	}
	if upstream != "origin/main" {
		t.Fatalf("upstream = %q, want origin/main", upstream)
	}
	if head != "94a67ef" {
		t.Fatalf("head = %q, want 94a67ef", head)
	}
	if detached {
		t.Fatalf("detached = true, want false")
	}
	if headMissing {
		t.Fatalf("headMissing = true, want false")
	}
	if !dirty {
		t.Fatalf("dirty = false, want true")
	}
	if untracked != 1 {
		t.Fatalf("untracked = %d, want 1", untracked)
	}
	if staged != 0 {
		t.Fatalf("staged = %d, want 0", staged)
	}
	if unstaged != 1 {
		t.Fatalf("unstaged = %d, want 1", unstaged)
	}
	if unmerged != 1 {
		t.Fatalf("unmerged = %d, want 1", unmerged)
	}
	if ahead != 2 {
		t.Fatalf("ahead = %d, want 2", ahead)
	}
	if behind != 1 {
		t.Fatalf("behind = %d, want 1", behind)
	}
}

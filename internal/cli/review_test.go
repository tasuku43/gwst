package cli

import "testing"

func TestParsePRURLGitHub(t *testing.T) {
	req, err := parsePRURL("https://github.com/owner/repo/pull/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Provider != "github" || req.Host != "github.com" || req.Owner != "owner" || req.Repo != "repo" || req.Number != 123 {
		t.Fatalf("unexpected result: %+v", req)
	}
}

func TestParsePRURLGitLab(t *testing.T) {
	req, err := parsePRURL("https://gitlab.com/owner/repo/-/merge_requests/45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Provider != "gitlab" || req.Host != "gitlab.com" || req.Owner != "owner" || req.Repo != "repo" || req.Number != 45 {
		t.Fatalf("unexpected result: %+v", req)
	}
}

func TestParsePRURLGitLabNoDash(t *testing.T) {
	req, err := parsePRURL("https://gitlab.com/owner/repo/merge_requests/45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Provider != "gitlab" || req.Owner != "owner" || req.Repo != "repo" || req.Number != 45 {
		t.Fatalf("unexpected result: %+v", req)
	}
}

func TestParsePRURLBitbucket(t *testing.T) {
	req, err := parsePRURL("https://bitbucket.org/owner/repo/pull-requests/7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Provider != "bitbucket" || req.Host != "bitbucket.org" || req.Owner != "owner" || req.Repo != "repo" || req.Number != 7 {
		t.Fatalf("unexpected result: %+v", req)
	}
}

func TestParsePRURLUnsupported(t *testing.T) {
	if _, err := parsePRURL("https://github.com/owner/repo/issues/1"); err == nil {
		t.Fatalf("expected error for non PR URL")
	}
	if _, err := parsePRURL("https://example.com/foo/bar"); err == nil {
		t.Fatalf("expected error for unsupported host/path")
	}
	if _, err := parsePRURL("https://gitlab.com/group/sub/repo/-/merge_requests/1"); err == nil {
		t.Fatalf("expected error for nested groups (not supported)")
	}
}

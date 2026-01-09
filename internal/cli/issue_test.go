package cli

import "testing"

func TestParseIssueURLGitHub(t *testing.T) {
	req, err := parseIssueURL("https://github.com/owner/repo/issues/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Provider != "github" || req.Host != "github.com" || req.Owner != "owner" || req.Repo != "repo" || req.Number != 123 {
		t.Fatalf("unexpected result: %+v", req)
	}
}

func TestParseIssueURLGitLab(t *testing.T) {
	req, err := parseIssueURL("https://gitlab.com/owner/repo/-/issues/45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Provider != "gitlab" || req.Host != "gitlab.com" || req.Owner != "owner" || req.Repo != "repo" || req.Number != 45 {
		t.Fatalf("unexpected result: %+v", req)
	}
}

func TestParseIssueURLGitLabNoDash(t *testing.T) {
	req, err := parseIssueURL("https://gitlab.com/owner/repo/issues/45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Provider != "gitlab" || req.Owner != "owner" || req.Repo != "repo" || req.Number != 45 {
		t.Fatalf("unexpected result: %+v", req)
	}
}

func TestParseIssueURLBitbucket(t *testing.T) {
	req, err := parseIssueURL("https://bitbucket.org/owner/repo/issues/7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Provider != "bitbucket" || req.Host != "bitbucket.org" || req.Owner != "owner" || req.Repo != "repo" || req.Number != 7 {
		t.Fatalf("unexpected result: %+v", req)
	}
}

func TestParseIssueURLUnsupported(t *testing.T) {
	if _, err := parseIssueURL("https://github.com/owner/repo/pull/1"); err == nil {
		t.Fatalf("expected error for non-issue URL")
	}
	if _, err := parseIssueURL("https://gitlab.com/group/sub/repo/-/issues/1"); err == nil {
		t.Fatalf("expected error for nested groups (not supported)")
	}
}

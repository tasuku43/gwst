package cli

import "testing"

func TestParseGitHubIssues(t *testing.T) {
	data := []byte(`[
  {"number": 1, "title": "Fix bug"},
  {"number": 2, "title": "PR", "pull_request": {}},
  {"number": 3, "title": "  Add feature  "}
]`)
	issues, err := parseGitHubIssues(data)
	if err != nil {
		t.Fatalf("parseGitHubIssues error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Number != 1 || issues[0].Title != "Fix bug" {
		t.Fatalf("unexpected first issue: %+v", issues[0])
	}
	if issues[1].Number != 3 || issues[1].Title != "Add feature" {
		t.Fatalf("unexpected second issue: %+v", issues[1])
	}
}

func TestParseGitHubIssuesInvalidJSON(t *testing.T) {
	if _, err := parseGitHubIssues([]byte(`{`)); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

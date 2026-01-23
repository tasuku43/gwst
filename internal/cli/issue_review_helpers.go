package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/infra/debuglog"
	"github.com/tasuku43/gwst/internal/ui"
)

type issueRepoChoice struct {
	Label    string
	Value    string
	Provider string
	Host     string
	Owner    string
	Repo     string
}

type issueSummary struct {
	Number int
	Title  string
}

func buildIssueRepoChoices(rootDir string) ([]issueRepoChoice, error) {
	repos, _, err := repo.List(rootDir)
	if err != nil {
		return nil, err
	}
	var choices []issueRepoChoice
	for _, entry := range repos {
		repoKey := displayRepoKey(entry.RepoKey)
		parts := strings.Split(repoKey, "/")
		if len(parts) < 3 {
			continue
		}
		host := parts[0]
		if !isGitHubHost(host) {
			continue
		}
		owner := parts[1]
		repoName := parts[2]
		label := fmt.Sprintf("%s (%s)", repoName, repoKey)
		value := repoSpecFromKey(entry.RepoKey)
		choices = append(choices, issueRepoChoice{
			Label:    label,
			Value:    value,
			Provider: "github",
			Host:     host,
			Owner:    owner,
			Repo:     repoName,
		})
	}
	return choices, nil
}

func toIssuePromptChoices(choices []issueRepoChoice) ([]ui.PromptChoice, map[string]issueRepoChoice) {
	prompt := make([]ui.PromptChoice, 0, len(choices))
	byValue := make(map[string]issueRepoChoice, len(choices))
	for _, choice := range choices {
		prompt = append(prompt, ui.PromptChoice{Label: choice.Label, Value: choice.Value})
		byValue[choice.Value] = choice
	}
	return prompt, byValue
}

func buildIssueChoices(issues []issueSummary) []ui.PromptChoice {
	var choices []ui.PromptChoice
	for _, issue := range issues {
		label := fmt.Sprintf("#%d", issue.Number)
		if strings.TrimSpace(issue.Title) != "" {
			label = fmt.Sprintf("#%d %s", issue.Number, strings.TrimSpace(issue.Title))
		}
		choices = append(choices, ui.PromptChoice{
			Label: label,
			Value: strconv.Itoa(issue.Number),
		})
	}
	return choices
}

type reviewRepoChoice struct {
	Label    string
	Value    string
	Provider string
	Host     string
	Owner    string
	Repo     string
	RepoURL  string
}

type prSummary struct {
	Number   int
	Title    string
	HeadRef  string
	HeadRepo string
	BaseRepo string
}

func buildReviewRepoChoices(rootDir string) ([]reviewRepoChoice, error) {
	repos, _, err := repo.List(rootDir)
	if err != nil {
		return nil, err
	}
	var choices []reviewRepoChoice
	for _, entry := range repos {
		repoKey := displayRepoKey(entry.RepoKey)
		parts := strings.Split(repoKey, "/")
		if len(parts) < 3 {
			continue
		}
		host := parts[0]
		owner := parts[1]
		repoName := parts[2]
		if !isGitHubHost(host) {
			continue
		}
		label := fmt.Sprintf("%s (%s/%s)", repoName, owner, repoName)
		repoURL := buildRepoURLFromParts(host, owner, repoName)
		value := repoSpecFromKey(entry.RepoKey)
		choices = append(choices, reviewRepoChoice{
			Label:    label,
			Value:    value,
			Provider: "github",
			Host:     host,
			Owner:    owner,
			Repo:     repoName,
			RepoURL:  repoURL,
		})
	}
	return choices, nil
}

func toPromptChoices(choices []reviewRepoChoice) ([]ui.PromptChoice, map[string]reviewRepoChoice) {
	prompt := make([]ui.PromptChoice, 0, len(choices))
	byValue := make(map[string]reviewRepoChoice, len(choices))
	for _, choice := range choices {
		prompt = append(prompt, ui.PromptChoice{Label: choice.Label, Value: choice.Value})
		byValue[choice.Value] = choice
	}
	return prompt, byValue
}

func buildPRChoices(prs []prSummary) []ui.PromptChoice {
	var choices []ui.PromptChoice
	for _, pr := range prs {
		label := fmt.Sprintf("#%d", pr.Number)
		if strings.TrimSpace(pr.Title) != "" {
			label = fmt.Sprintf("#%d %s", pr.Number, strings.TrimSpace(pr.Title))
		}
		choices = append(choices, ui.PromptChoice{
			Label: label,
			Value: encodeReviewSelection(pr),
		})
	}
	return choices
}

func isGitHubHost(host string) bool {
	lower := strings.ToLower(strings.TrimSpace(host))
	return strings.Contains(lower, "github")
}

type githubIssueItem struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	PullRequest json.RawMessage `json:"pull_request"`
}

func runExternalCommand(ctx context.Context, name string, args []string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	trace := ""
	if debuglog.Enabled() {
		trace = debuglog.NewTrace("exec")
		debuglog.LogCommand(trace, debuglog.FormatCommand(name, args))
	}
	err := cmd.Run()
	if debuglog.Enabled() {
		debuglog.LogStdoutLines(trace, stdout.String())
		debuglog.LogStderrLines(trace, stderr.String())
		debuglog.LogExit(trace, debuglog.ExitCode(err))
	}
	return stdout.String(), stderr.String(), err
}

func fetchGitHubIssues(ctx context.Context, host, owner, repoName string) ([]issueSummary, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" {
		return nil, fmt.Errorf("owner/repo is required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/issues", owner, repoName)
	args := []string{"api", "-X", "GET", endpoint, "-f", "state=open", "-f", "sort=updated", "-f", "direction=desc", "-f", "per_page=50"}
	if host != "" && !strings.EqualFold(host, "github.com") {
		args = append([]string{"api", "--hostname", host}, args[1:]...)
	}
	stdout, stderr, err := runExternalCommand(ctx, "gh", args)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg != "" {
			return nil, fmt.Errorf("gh api failed: %s", msg)
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}
	return parseGitHubIssues([]byte(stdout))
}

func fetchGitHubIssue(ctx context.Context, host, owner, repoName string, number int) (issueSummary, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" || number <= 0 {
		return issueSummary{}, fmt.Errorf("owner/repo and issue number are required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/issues/%d", owner, repoName, number)
	args := []string{"api", "-X", "GET", endpoint}
	if host != "" && !strings.EqualFold(host, "github.com") {
		args = append([]string{"api", "--hostname", host}, args[1:]...)
	}
	stdout, stderr, err := runExternalCommand(ctx, "gh", args)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg != "" {
			return issueSummary{}, fmt.Errorf("gh api failed: %s", msg)
		}
		return issueSummary{}, fmt.Errorf("gh api failed: %w", err)
	}
	var item githubIssueItem
	if err := json.Unmarshal([]byte(stdout), &item); err != nil {
		return issueSummary{}, fmt.Errorf("parse gh api response: %w", err)
	}
	if item.Number == 0 {
		return issueSummary{}, fmt.Errorf("issue not found")
	}
	return issueSummary{
		Number: item.Number,
		Title:  strings.TrimSpace(item.Title),
	}, nil
}

func parseGitHubIssues(data []byte) ([]issueSummary, error) {
	var raw []githubIssueItem
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gh api response: %w", err)
	}
	var issues []issueSummary
	for _, item := range raw {
		if item.Number == 0 {
			continue
		}
		if len(item.PullRequest) != 0 {
			continue
		}
		issues = append(issues, issueSummary{
			Number: item.Number,
			Title:  strings.TrimSpace(item.Title),
		})
	}
	return issues, nil
}

type githubPRItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Head   struct {
		Ref  string `json:"ref"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
}

func fetchGitHubPR(ctx context.Context, host, owner, repoName string, number int) (prSummary, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" || number <= 0 {
		return prSummary{}, fmt.Errorf("owner/repo and PR number are required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repoName, number)
	args := []string{"api", "-X", "GET", endpoint}
	if host != "" && !strings.EqualFold(host, "github.com") {
		args = append([]string{"api", "--hostname", host}, args[1:]...)
	}
	stdout, stderr, err := runExternalCommand(ctx, "gh", args)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg != "" {
			return prSummary{}, fmt.Errorf("gh api failed: %s", msg)
		}
		return prSummary{}, fmt.Errorf("gh api failed: %w", err)
	}
	var item githubPRItem
	if err := json.Unmarshal([]byte(stdout), &item); err != nil {
		return prSummary{}, fmt.Errorf("parse gh api response: %w", err)
	}
	return normalizeGitHubPR(item), nil
}

func fetchGitHubPRs(ctx context.Context, host, owner, repoName string) ([]prSummary, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" {
		return nil, fmt.Errorf("owner/repo is required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/pulls", owner, repoName)
	args := []string{"api", "-X", "GET", endpoint, "-f", "state=open", "-f", "sort=updated", "-f", "direction=desc", "-f", "per_page=50"}
	if host != "" && !strings.EqualFold(host, "github.com") {
		args = append([]string{"api", "--hostname", host}, args[1:]...)
	}
	stdout, stderr, err := runExternalCommand(ctx, "gh", args)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg != "" {
			return nil, fmt.Errorf("gh api failed: %s", msg)
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}
	return parseGitHubPRs([]byte(stdout))
}

func parseGitHubPRs(data []byte) ([]prSummary, error) {
	var raw []githubPRItem
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gh api response: %w", err)
	}
	var prs []prSummary
	for _, item := range raw {
		if item.Number == 0 {
			continue
		}
		prs = append(prs, normalizeGitHubPR(item))
	}
	return prs, nil
}

func normalizeGitHubPR(item githubPRItem) prSummary {
	return prSummary{
		Number:   item.Number,
		Title:    strings.TrimSpace(item.Title),
		HeadRef:  strings.TrimSpace(item.Head.Ref),
		HeadRepo: strings.TrimSpace(item.Head.Repo.FullName),
		BaseRepo: strings.TrimSpace(item.Base.Repo.FullName),
	}
}

func encodeReviewSelection(pr prSummary) string {
	escape := url.QueryEscape
	return strings.Join([]string{
		strconv.Itoa(pr.Number),
		escape(pr.HeadRef),
		escape(pr.HeadRepo),
		escape(pr.BaseRepo),
		escape(pr.Title),
	}, "|")
}

func decodeReviewSelection(value string) (prSummary, error) {
	parts := strings.Split(value, "|")
	if len(parts) == 1 {
		num, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return prSummary{}, fmt.Errorf("invalid PR selection: %s", value)
		}
		return prSummary{}, fmt.Errorf("missing PR metadata for #%d; re-run selection", num)
	}
	if len(parts) != 5 {
		return prSummary{}, fmt.Errorf("invalid PR selection: %s", value)
	}
	num, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return prSummary{}, fmt.Errorf("invalid PR number: %s", parts[0])
	}
	unescape := func(v string) string {
		out, err := url.QueryUnescape(v)
		if err != nil {
			return v
		}
		return out
	}
	return prSummary{
		Number:   num,
		HeadRef:  strings.TrimSpace(unescape(parts[1])),
		HeadRepo: strings.TrimSpace(unescape(parts[2])),
		BaseRepo: strings.TrimSpace(unescape(parts[3])),
		Title:    strings.TrimSpace(unescape(parts[4])),
	}, nil
}

func splitRepoFullName(fullName string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(fullName), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func formatReviewWorkspaceID(owner, repo string, number int) string {
	return fmt.Sprintf("%s-%s-REVIEW-PR-%d", strings.ToUpper(strings.TrimSpace(owner)), strings.ToUpper(strings.TrimSpace(repo)), number)
}

func formatIssueWorkspaceID(owner, repo string, number int) string {
	return fmt.Sprintf("%s-%s-ISSUE-%d", strings.ToUpper(strings.TrimSpace(owner)), strings.ToUpper(strings.TrimSpace(repo)), number)
}

type issueRequest struct {
	Provider string
	Host     string
	Owner    string
	Repo     string
	Number   int
}

func parseIssueURL(raw string) (issueRequest, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return issueRequest{}, fmt.Errorf("invalid issue URL: %w", err)
	}
	host := strings.TrimSpace(u.Hostname())
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return issueRequest{}, fmt.Errorf("invalid issue URL path: %s", u.Path)
	}

	for i := 0; i < len(parts)-1; i++ {
		if parts[i] != "issues" {
			continue
		}
		num, err := strconv.Atoi(parts[i+1])
		if err != nil {
			return issueRequest{}, fmt.Errorf("invalid issue number: %s", parts[i+1])
		}
		repoIdx := i - 1
		if repoIdx >= 1 && parts[repoIdx] == "-" {
			repoIdx--
		}
		if repoIdx < 1 {
			return issueRequest{}, fmt.Errorf("invalid issue URL path: %s", u.Path)
		}
		ownerParts := parts[:repoIdx]
		provider := issueProvider(host, repoIdx, i)
		if provider == "gitlab" {
			if len(ownerParts) != 1 {
				return issueRequest{}, fmt.Errorf("nested groups are not supported: %s", strings.Join(ownerParts, "/"))
			}
		} else if len(ownerParts) != 1 {
			return issueRequest{}, fmt.Errorf("invalid issue URL path: %s", u.Path)
		}
		return issueRequest{
			Provider: provider,
			Host:     host,
			Owner:    ownerParts[0],
			Repo:     parts[repoIdx],
			Number:   num,
		}, nil
	}

	return issueRequest{}, fmt.Errorf("unsupported issue URL: %s", raw)
}

func issueProvider(host string, repoIdx, issueIdx int) string {
	lowerHost := strings.ToLower(strings.TrimSpace(host))
	if repoIdx < issueIdx-1 || strings.Contains(lowerHost, "gitlab") {
		return "gitlab"
	}
	if strings.Contains(lowerHost, "bitbucket") {
		return "bitbucket"
	}
	return "github"
}

type prRequest struct {
	Provider string
	Host     string
	Owner    string
	Repo     string
	Number   int
}

func parsePRURL(raw string) (prRequest, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return prRequest{}, fmt.Errorf("invalid PR/MR URL: %w", err)
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return prRequest{}, fmt.Errorf("invalid PR URL host: %s", raw)
	}
	if !isGitHubHost(host) {
		return prRequest{}, fmt.Errorf("unsupported PR host: %s", host)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return prRequest{}, fmt.Errorf("invalid PR/MR URL path: %s", u.Path)
	}

	// GitHub style: /owner/repo/pull/123
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "pull" && i >= 2 {
			num, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return prRequest{}, fmt.Errorf("invalid PR number: %s", parts[i+1])
			}
			return prRequest{
				Provider: "github",
				Host:     host,
				Owner:    parts[i-2],
				Repo:     parts[i-1],
				Number:   num,
			}, nil
		}
	}

	return prRequest{}, fmt.Errorf("unsupported PR/MR URL: %s", raw)
}

func buildRepoURLFromParts(host, owner, repoName string) string {
	repoName = strings.TrimSuffix(repoName, ".git")
	switch strings.ToLower(strings.TrimSpace(defaultRepoProtocol)) {
	case "https":
		return fmt.Sprintf("https://%s/%s/%s.git", host, owner, repoName)
	default:
		return fmt.Sprintf("git@%s:%s/%s.git", host, owner, repoName)
	}
}

func buildIssueURLFromParts(host, owner, repoName string, number int) string {
	repoName = strings.TrimSuffix(repoName, ".git")
	return fmt.Sprintf("https://%s/%s/%s/issues/%d", host, owner, repoName, number)
}

func buildPRURLFromParts(host, owner, repoName string, number int) string {
	repoName = strings.TrimSuffix(repoName, ".git")
	return fmt.Sprintf("https://%s/%s/%s/pull/%d", host, owner, repoName, number)
}

func issueTitleFromLabel(label string, number int) string {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return ""
	}
	prefix := fmt.Sprintf("#%d", number)
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	title := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	return title
}

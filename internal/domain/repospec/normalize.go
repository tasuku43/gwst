package repospec

import (
	"fmt"
	"net/url"
	"strings"
)

type Spec struct {
	Host    string
	Owner   string
	Repo    string
	RepoKey string
}

func Normalize(input string) (Spec, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return Spec{}, fmt.Errorf("repo spec is empty")
	}

	var host string
	var path string

	switch {
	case strings.HasPrefix(trimmed, "git@"):
		at := strings.Index(trimmed, "@")
		colon := strings.Index(trimmed, ":")
		if at < 0 || colon < 0 || colon < at {
			return Spec{}, fmt.Errorf("invalid ssh repo spec: %q", input)
		}
		host = trimmed[at+1 : colon]
		path = trimmed[colon+1:]
	case strings.HasPrefix(trimmed, "https://"):
		u, err := url.Parse(trimmed)
		if err != nil {
			return Spec{}, fmt.Errorf("invalid https repo spec: %q", input)
		}
		host = u.Hostname()
		path = strings.TrimPrefix(u.Path, "/")
	case strings.HasPrefix(trimmed, "file://"):
		u, err := url.Parse(trimmed)
		if err != nil {
			return Spec{}, fmt.Errorf("invalid file repo spec: %q", input)
		}
		// For file remotes, infer <host>/<owner>/<repo> from the tail of the path.
		// Expected: file:///.../<host>/<owner>/<repo>(.git)
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 3 {
			return Spec{}, fmt.Errorf("file repo spec must end with <host>/<owner>/<repo>: %q", input)
		}
		host = parts[len(parts)-3]
		owner := parts[len(parts)-2]
		repo := parts[len(parts)-1]
		path = fmt.Sprintf("%s/%s", owner, repo)
	default:
		return Spec{}, fmt.Errorf("repo spec must be ssh, https, or file: %q", input)
	}

	owner, repo, err := splitOwnerRepo(path)
	if err != nil {
		return Spec{}, err
	}
	if host == "" {
		return Spec{}, fmt.Errorf("host is required in repo spec: %q", input)
	}

	spec := Spec{
		Host:    host,
		Owner:   owner,
		Repo:    repo,
		RepoKey: fmt.Sprintf("%s/%s/%s", host, owner, repo),
	}
	return spec, nil
}

func splitOwnerRepo(path string) (string, string, error) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", "", fmt.Errorf("repo path is empty")
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("repo path must be <owner>/<repo>")
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("owner/repo cannot be empty")
	}

	return owner, repo, nil
}

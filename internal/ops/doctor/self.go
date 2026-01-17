package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/tasuku43/gwst/internal/core/gitcmd"
)

type SelfResult struct {
	Issues   []Issue
	Warnings []string
	Details  []string
}

type gitVersion struct {
	major int
	minor int
	patch int
}

func (v gitVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func (v gitVersion) Less(other gitVersion) bool {
	if v.major != other.major {
		return v.major < other.major
	}
	if v.minor != other.minor {
		return v.minor < other.minor
	}
	return v.patch < other.patch
}

var minGitVersion = gitVersion{major: 2, minor: 20, patch: 0}
var gitVersionPattern = regexp.MustCompile(`\b(\d+)\.(\d+)(?:\.(\d+))?`)

func SelfCheck(ctx context.Context) (SelfResult, error) {
	result := SelfResult{
		Details: []string{
			fmt.Sprintf("os: %s/%s", runtime.GOOS, runtime.GOARCH),
			fmt.Sprintf("minimum git version: %s", minGitVersion.String()),
		},
	}

	result.Warnings = append(result.Warnings, osCaveats(runtime.GOOS)...)

	gitPath, err := exec.LookPath("git")
	if err != nil {
		result.Issues = append(result.Issues, Issue{
			Kind:    "missing_dependency",
			Message: "git not found in PATH",
		})
		result.Details = append(result.Details, "git: not found")
		return result, nil
	}
	result.Details = append(result.Details, fmt.Sprintf("git path: %s", gitPath))

	versionOutput, err := readGitVersion(ctx)
	if err != nil {
		result.Issues = append(result.Issues, Issue{
			Kind:    "git_version_check_failed",
			Message: err.Error(),
		})
		result.Details = append(result.Details, "git version: unknown")
		return result, nil
	}
	result.Details = append(result.Details, fmt.Sprintf("git version: %s", versionOutput))

	parsed, ok := parseGitVersion(versionOutput)
	if !ok {
		result.Issues = append(result.Issues, Issue{
			Kind:    "invalid_git_version",
			Message: fmt.Sprintf("unable to parse git version: %s", versionOutput),
		})
		return result, nil
	}
	if parsed.Less(minGitVersion) {
		result.Issues = append(result.Issues, Issue{
			Kind:    "git_version_too_old",
			Message: fmt.Sprintf("git %s is older than required %s", parsed.String(), minGitVersion.String()),
		})
	}

	return result, nil
}

func readGitVersion(ctx context.Context) (string, error) {
	res, err := gitcmd.Run(ctx, []string{"version"}, gitcmd.Options{})
	if err != nil {
		if strings.TrimSpace(res.Stderr) != "" {
			return "", fmt.Errorf("git version failed: %s", strings.TrimSpace(res.Stderr))
		}
		return "", fmt.Errorf("git version failed: %w", err)
	}
	out := strings.TrimSpace(res.Stdout)
	if out == "" {
		out = strings.TrimSpace(res.Stderr)
	}
	if out == "" {
		return "", fmt.Errorf("git version returned no output")
	}
	return out, nil
}

func parseGitVersion(output string) (gitVersion, bool) {
	matches := gitVersionPattern.FindStringSubmatch(output)
	if len(matches) < 3 {
		return gitVersion{}, false
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return gitVersion{}, false
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return gitVersion{}, false
	}
	patch := 0
	if len(matches) > 3 && matches[3] != "" {
		value, err := strconv.Atoi(matches[3])
		if err != nil {
			return gitVersion{}, false
		}
		patch = value
	}
	return gitVersion{major: major, minor: minor, patch: patch}, true
}

func osCaveats(goos string) []string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "windows":
		return []string{"Windows detected: behavior may be limited; consider using WSL if issues occur."}
	default:
		return nil
	}
}

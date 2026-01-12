package workspace

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tasuku43/gws/internal/core/gitcmd"
	"github.com/tasuku43/gws/internal/core/paths"
)

type StatusResult struct {
	WorkspaceID string
	Repos       []RepoStatus
	Warnings    []error
}

type RepoStatus struct {
	Alias          string
	Branch         string
	Upstream       string
	Head           string
	Dirty          bool
	UntrackedCount int
	StagedCount    int
	UnstagedCount  int
	UnmergedCount  int
	AheadCount     int
	BehindCount    int
	WorktreePath   string
	RawStatus      string
	Error          error
}

func Status(ctx context.Context, rootDir, workspaceID string) (StatusResult, error) {
	if workspaceID == "" {
		return StatusResult{}, fmt.Errorf("workspace id is required")
	}
	if rootDir == "" {
		return StatusResult{}, fmt.Errorf("root directory is required")
	}

	wsDir := WorkspaceDir(rootDir, workspaceID)
	if exists, err := paths.DirExists(wsDir); err != nil {
		return StatusResult{}, err
	} else if !exists {
		return StatusResult{}, fmt.Errorf("workspace does not exist: %s", wsDir)
	}

	repos, warnings, err := ScanRepos(ctx, wsDir)
	if err != nil {
		return StatusResult{}, err
	}

	result := StatusResult{
		WorkspaceID: workspaceID,
		Warnings:    warnings,
	}
	for _, repo := range repos {
		repoStatus := RepoStatus{
			Alias:        repo.Alias,
			Branch:       repo.Branch,
			WorktreePath: repo.WorktreePath,
		}

		statusOut, statusErr := gitStatusPorcelain(ctx, repo.WorktreePath)
		if statusErr != nil {
			repoStatus.Error = statusErr
			result.Repos = append(result.Repos, repoStatus)
			continue
		}

		repoStatus.RawStatus = statusOut
		repoStatus.Branch, repoStatus.Upstream, repoStatus.Head, repoStatus.Dirty, repoStatus.UntrackedCount, repoStatus.StagedCount, repoStatus.UnstagedCount, repoStatus.UnmergedCount, repoStatus.AheadCount, repoStatus.BehindCount = parseStatusPorcelainV2(statusOut, repoStatus.Branch)
		result.Repos = append(result.Repos, repoStatus)
	}

	return result, nil
}

func gitStatusPorcelain(ctx context.Context, worktreePath string) (string, error) {
	return gitcmd.StatusPorcelainV2(ctx, worktreePath)
}

func parseStatusPorcelainV2(output, fallbackBranch string) (string, string, string, bool, int, int, int, int, int, int) {
	branch := fallbackBranch
	var upstream string
	var head string
	var dirty bool
	var untracked int
	var staged int
	var unstaged int
	var unmerged int
	var ahead int
	var behind int

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			switch fields[1] {
			case "branch.oid":
				if fields[2] != "(initial)" {
					head = shortSHA(fields[2])
				}
			case "branch.head":
				if fields[2] != "(detached)" && fields[2] != "(unknown)" {
					branch = fields[2]
				}
			case "branch.upstream":
				if fields[2] != "(unknown)" {
					upstream = fields[2]
				}
			case "branch.ab":
				for _, field := range fields[2:] {
					if strings.HasPrefix(field, "+") {
						ahead = parseCount(field[1:])
					}
					if strings.HasPrefix(field, "-") {
						behind = parseCount(field[1:])
					}
				}
			}
			continue
		}

		if strings.HasPrefix(line, "? ") {
			untracked++
			dirty = true
			continue
		}

		if strings.HasPrefix(line, "u ") {
			unmerged++
			dirty = true
			continue
		}
		if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				xy := fields[1]
				if len(xy) >= 2 {
					if xy[0] != '.' {
						staged++
					}
					if xy[1] != '.' {
						unstaged++
					}
					if xy[0] != '.' || xy[1] != '.' {
						dirty = true
					}
				}
			}
			continue
		}
		dirty = true
	}

	return branch, upstream, head, dirty, untracked, staged, unstaged, unmerged, ahead, behind
}

func shortSHA(oid string) string {
	if len(oid) <= 7 {
		return oid
	}
	return oid[:7]
}

func parseCount(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	if n < 0 {
		return 0
	}
	return n
}

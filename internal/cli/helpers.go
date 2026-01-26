package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gion/internal/app/manifestplan"
	"github.com/tasuku43/gion/internal/domain/repo"
	"github.com/tasuku43/gion/internal/domain/workspace"
	"github.com/tasuku43/gion/internal/infra/output"
	"github.com/tasuku43/gion/internal/ui"
)

func displayRepoKey(repoKey string) string {
	display := strings.TrimSuffix(repoKey, ".git")
	display = strings.TrimSuffix(display, "/")
	return display
}

func displayPresetRepo(repoSpec string) string {
	return repo.DisplaySpec(repoSpec)
}

func displayRepoSpec(repoSpec string) string {
	return displayPresetRepo(repoSpec)
}

func displayRepoName(repoSpec string) string {
	return repo.DisplayName(repoSpec)
}

func repoDestForSpec(rootDir, repoSpec string) string {
	store := repoStoreRel(rootDir, repoSpec)
	return store
}

func repoStoreRel(rootDir, repoSpec string) string {
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		return ""
	}
	storePath := repo.StorePath(rootDir, spec)
	return relPath(rootDir, storePath)
}

func worktreeDest(rootDir, workspaceID, repoSpec string) string {
	spec, _, err := repo.Normalize(repoSpec)
	if err != nil || spec.Repo == "" {
		return ""
	}
	wsPath := workspace.WorktreePath(rootDir, workspaceID, spec.Repo)
	return relPath(rootDir, wsPath)
}

func relPath(rootDir, path string) string {
	if strings.TrimSpace(rootDir) == "" || strings.TrimSpace(path) == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func formatStep(action, target, dest string) string {
	parts := []string{strings.TrimSpace(action)}
	if strings.TrimSpace(target) != "" {
		parts = append(parts, truncateMiddle(strings.TrimSpace(target), 80))
	}
	text := strings.Join(parts, " ")
	if strings.TrimSpace(dest) != "" {
		return fmt.Sprintf("%s -> %s", text, truncateMiddle(dest, 80))
	}
	return text
}

func formatStepWithIndex(action, target, dest string, index, total int) string {
	if total > 0 {
		if strings.TrimSpace(target) != "" {
			target = fmt.Sprintf("%s (%d/%d)", target, index, total)
		} else {
			action = fmt.Sprintf("%s (%d/%d)", action, index, total)
		}
	}
	return formatStep(action, target, dest)
}

func truncateMiddle(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if max <= 0 || len(trimmed) <= max {
		return trimmed
	}
	if max < 10 {
		return trimmed[:max]
	}
	keep := (max - 3) / 2
	return fmt.Sprintf("%s...%s", trimmed[:keep], trimmed[len(trimmed)-keep:])
}

func startSteps(renderer *ui.Renderer) {
	if renderer == nil {
		return
	}
	renderer.Section("Steps")
}

func ensureRepoGet(ctx context.Context, rootDir string, repoSpecs []string, noPrompt bool, theme ui.Theme, useColor bool) error {
	if len(repoSpecs) == 0 {
		return nil
	}
	var missing []string
	for _, repoSpec := range repoSpecs {
		if strings.TrimSpace(repoSpec) == "" {
			continue
		}
		missing = append(missing, repoSpec)
	}
	if len(missing) == 0 {
		return nil
	}
	if noPrompt {
		return fmt.Errorf("repo get required for: %s", strings.Join(missing, ", "))
	}
	label := "repos"
	if len(missing) == 1 {
		label = "repo"
	}
	output.Step(fmt.Sprintf("repo get required for %d %s", len(missing), label))
	for _, repoSpec := range missing {
		output.Log(fmt.Sprintf("gion repo get %s", displayRepoSpec(repoSpec)))
	}
	confirm, err := ui.PromptConfirmInline("run now?", theme, useColor)
	if err != nil {
		return err
	}
	if !confirm {
		return fmt.Errorf("repo get required for: %s", strings.Join(missing, ", "))
	}
	for i, repoSpec := range missing {
		output.Step(formatStepWithIndex("repo get", displayRepoSpec(repoSpec), repoDestForSpec(rootDir, repoSpec), i+1, len(missing)))
		if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
			return err
		}
	}
	return nil
}

func repoSpecFromKey(repoKey string) string {
	trimmed := strings.TrimSuffix(repoKey, ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return repoKey
	}
	host := parts[0]
	owner := parts[1]
	repoName := parts[2]
	if strings.EqualFold(strings.TrimSpace(defaultRepoProtocol), "ssh") {
		return fmt.Sprintf("git@%s:%s/%s.git", host, owner, repoName)
	}
	return fmt.Sprintf("https://%s/%s/%s.git", host, owner, repoName)
}

func formatRepoName(alias, repoKey string) string {
	name := strings.TrimSpace(alias)
	if name != "" {
		return name
	}
	return repoKey
}

func formatRepoLabel(name, branch string) string {
	if strings.TrimSpace(branch) != "" {
		return fmt.Sprintf("%s (branch: %s)", name, branch)
	}
	return name
}

func appendWarningLines(lines []string, prefix string, warnings []error) []string {
	for _, warning := range warnings {
		message := compactError(warning)
		if strings.TrimSpace(prefix) != "" {
			message = fmt.Sprintf("%s: %s", prefix, message)
		}
		lines = append(lines, message)
	}
	return lines
}

func compactError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "unknown error"
	}
	return strings.Join(strings.Fields(msg), " ")
}

func splitNonEmptyLines(text string) []string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func planWorkspaceDescription(plan manifestplan.Result, workspaceID string) string {
	if ws, ok := plan.Actual.Workspaces[workspaceID]; ok {
		return ws.Description
	}
	if ws, ok := plan.Desired.Workspaces[workspaceID]; ok {
		return ws.Description
	}
	return ""
}

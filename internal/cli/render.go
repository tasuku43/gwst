package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/tasuku43/gwst/internal/app/doctor"
	"github.com/tasuku43/gwst/internal/app/initcmd"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/preset"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/output"
	"github.com/tasuku43/gwst/internal/ui"
)

type statusDetail struct {
	text string
	warn bool
}

type treeLineStyle int

const (
	treeLineNormal treeLineStyle = iota
	treeLineWarn
	treeLineError
	treeLineSuccess
	treeLineAccent
)

func renderWorkspaceRepos(r *ui.Renderer, repos []workspace.Repo, extraIndent string) {
	if r == nil {
		return
	}
	for i, repo := range repos {
		prefix := "├─ "
		if i == len(repos)-1 {
			prefix = "└─ "
		}
		name := formatRepoName(repo.Alias, repo.RepoKey)
		r.TreeLineBranchMuted(extraIndent+prefix, name, repo.Branch)
	}
}

func renderWorkspaceBlock(r *ui.Renderer, workspaceID, description string, repos []workspace.Repo) {
	if r == nil {
		return
	}
	r.BulletWithDescription(workspaceID, description, fmt.Sprintf("(repos: %d)", len(repos)))
	renderWorkspaceRepos(r, repos, output.Indent)
}

func buildStatusDetails(repo workspace.RepoStatus) []statusDetail {
	var details []statusDetail
	head := strings.TrimSpace(repo.Head)
	if head != "" {
		details = append(details, statusDetail{text: fmt.Sprintf("head: %s", head)})
	}
	if repo.StagedCount == 0 && repo.UnstagedCount == 0 && repo.UntrackedCount == 0 && repo.UnmergedCount == 0 {
		details = append(details, statusDetail{text: "changes: clean"})
		return details
	}
	if repo.StagedCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("staged: %d", repo.StagedCount), warn: true})
	}
	if repo.UnstagedCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("unstaged: %d", repo.UnstagedCount), warn: true})
	}
	if repo.UntrackedCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("untracked: %d", repo.UntrackedCount), warn: true})
	}
	if repo.UnmergedCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("unmerged: %d", repo.UnmergedCount), warn: true})
	}
	if repo.AheadCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("ahead: %d", repo.AheadCount), warn: true})
	}
	if repo.BehindCount > 0 {
		details = append(details, statusDetail{text: fmt.Sprintf("behind: %d", repo.BehindCount), warn: true})
	}
	return details
}

func issueDetails(issue doctor.Issue) []string {
	var details []string
	if strings.TrimSpace(issue.Path) != "" {
		details = append(details, fmt.Sprintf("path: %s", issue.Path))
	}
	if strings.TrimSpace(issue.Message) != "" {
		details = append(details, fmt.Sprintf("message: %s", issue.Message))
	}
	return details
}

func presetIssueDetails(issue preset.ValidationIssue, path string) []string {
	var details []string
	if strings.TrimSpace(path) != "" && (issue.Kind == preset.IssueKindFile || issue.Kind == preset.IssueKindInvalidYAML) {
		details = append(details, fmt.Sprintf("path: %s", path))
	}
	if strings.TrimSpace(issue.Preset) != "" {
		details = append(details, fmt.Sprintf("preset: %s", issue.Preset))
	}
	if strings.TrimSpace(issue.Repo) != "" {
		details = append(details, fmt.Sprintf("repo: %s", issue.Repo))
	}
	if strings.TrimSpace(issue.Message) != "" {
		details = append(details, fmt.Sprintf("message: %s", issue.Message))
	}
	return details
}

func renderTreeLines(r *ui.Renderer, lines []string, style treeLineStyle) {
	for i, line := range lines {
		prefix := "├─ "
		if i == len(lines)-1 {
			prefix = "└─ "
		}
		switch style {
		case treeLineWarn:
			r.TreeLineWarn(output.Indent+prefix, line)
		case treeLineError:
			r.TreeLineError(output.Indent+prefix, line)
		case treeLineSuccess:
			r.TreeLineSuccess(output.Indent+prefix, line)
		case treeLineAccent:
			r.TreeLineAccent(output.Indent+prefix, line)
		default:
			r.TreeLine(output.Indent+prefix, line)
		}
	}
}

func buildUnifiedDiffLines(current, next []byte) ([]string, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(current)),
		B:        difflib.SplitLines(string(next)),
		FromFile: "gwst.yaml (current)",
		ToFile:   "gwst.yaml (target)",
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return nil, err
	}
	lines := difflib.SplitLines(text)
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\n")
	}
	return lines, nil
}

func renderDiffLines(renderer *ui.Renderer, lines []string) {
	if renderer == nil || len(lines) == 0 {
		return
	}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"), strings.HasPrefix(line, "@@"):
			renderer.TreeLineAccent("", line)
		case strings.HasPrefix(line, "+"):
			renderer.TreeLineSuccess("", line)
		case strings.HasPrefix(line, "-"):
			renderer.TreeLineError("", line)
		default:
			renderer.TreeLine("", line)
		}
	}
}

func renderPlanChanges(renderer *ui.Renderer, changes []manifestplan.WorkspaceChange) {
	if renderer == nil {
		return
	}
	for _, change := range changes {
		switch change.Kind {
		case manifestplan.WorkspaceAdd:
			renderer.BulletSuccess(fmt.Sprintf("+ add workspace %s", change.WorkspaceID))
			renderPlanRepoChanges(renderer, change.Repos)
		case manifestplan.WorkspaceRemove:
			renderer.BulletError(fmt.Sprintf("- remove workspace %s", change.WorkspaceID))
		case manifestplan.WorkspaceUpdate:
			renderer.BulletAccent(fmt.Sprintf("~ update workspace %s", change.WorkspaceID))
			renderPlanRepoChanges(renderer, change.Repos)
		}
	}
}

func renderPlanRepoChanges(renderer *ui.Renderer, changes []manifestplan.RepoChange) {
	if renderer == nil || len(changes) == 0 {
		return
	}
	var lines []string
	var styles []treeLineStyle
	for _, change := range changes {
		switch change.Kind {
		case manifestplan.RepoAdd:
			lines = append(lines, fmt.Sprintf("+ add repo %s (%s) branch %s", change.Alias, change.ToRepo, change.ToBranch))
			styles = append(styles, treeLineSuccess)
		case manifestplan.RepoRemove:
			lines = append(lines, fmt.Sprintf("- remove repo %s (%s) branch %s", change.Alias, change.FromRepo, change.FromBranch))
			styles = append(styles, treeLineError)
		case manifestplan.RepoUpdate:
			lines = append(lines, formatPlanRepoUpdate(change))
			styles = append(styles, treeLineAccent)
		}
	}
	for i, line := range lines {
		style := treeLineNormal
		if i < len(styles) {
			style = styles[i]
		}
		renderTreeLines(renderer, []string{line}, style)
	}
}

func formatPlanRepoUpdate(change manifestplan.RepoChange) string {
	fromRepo := strings.TrimSpace(change.FromRepo)
	toRepo := strings.TrimSpace(change.ToRepo)
	fromBranch := strings.TrimSpace(change.FromBranch)
	toBranch := strings.TrimSpace(change.ToBranch)

	switch {
	case fromRepo == toRepo && fromBranch != toBranch:
		return fmt.Sprintf("~ update repo %s: branch %s -> %s", change.Alias, fromBranch, toBranch)
	case fromRepo != toRepo && fromBranch == toBranch:
		return fmt.Sprintf("~ update repo %s: repo %s -> %s", change.Alias, fromRepo, toRepo)
	default:
		return fmt.Sprintf("~ update repo %s: %s (%s) -> %s (%s)", change.Alias, fromRepo, fromBranch, toRepo, toBranch)
	}
}

func renderWarningsSection(r *ui.Renderer, title string, warnings []string, leadBlank bool) {
	if r == nil || len(warnings) == 0 {
		return
	}
	if leadBlank {
		r.Blank()
	}
	r.Section("Info")
	r.Bullet(title)
	renderTreeLines(r, warnings, treeLineWarn)
}

func renderSuggestions(r *ui.Renderer, useColor bool, lines []string) {
	if !useColor || r == nil {
		return
	}
	var filtered []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		return
	}
	r.Blank()
	r.Section("Suggestion")
	for _, line := range filtered {
		r.Bullet(line)
	}
}

func writeWorkspaceStatusText(result workspace.StatusResult) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	warningLines := appendWarningLines(nil, "", result.Warnings)
	for _, repo := range result.Repos {
		if repo.Error != nil {
			label := strings.TrimSpace(repo.Alias)
			if label == "" {
				label = filepath.Base(repo.WorktreePath)
			}
			warningLines = append(warningLines, fmt.Sprintf("%s: %v", label, repo.Error))
		}
	}
	if len(warningLines) > 0 {
		renderWarningsSection(renderer, "warnings", warningLines, false)
		renderer.Blank()
	}

	renderer.Section("Result")

	for _, repo := range result.Repos {
		label := repo.Alias
		if strings.TrimSpace(label) == "" {
			label = filepath.Base(repo.WorktreePath)
		}
		if strings.TrimSpace(repo.Branch) != "" {
			label = fmt.Sprintf("%s (branch: %s)", label, repo.Branch)
		}
		renderer.Bullet(label)

		details := buildStatusDetails(repo)
		for i, detail := range details {
			prefix := "├─ "
			if i == len(details)-1 {
				prefix = "└─ "
			}
			prefix = output.Indent + prefix
			if detail.warn {
				renderer.TreeLineWarn(prefix, detail.text)
			} else {
				renderer.TreeLineBranchMuted(prefix, detail.text, "")
			}
		}
	}
}

func writeWorkspaceListText(ctx context.Context, rootDir string, entries []workspace.Entry, warnings []error, showDetails bool) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	type workspaceListEntry struct {
		entry  workspace.Entry
		repos  []workspace.Repo
		state  workspace.WorkspaceState
		status workspace.StatusResult
	}
	var items []workspaceListEntry
	var repoWarnings []string
	for _, entry := range entries {
		repos, warnings, err := workspace.ScanRepos(ctx, entry.WorkspacePath)
		if err != nil {
			repoWarnings = append(repoWarnings, fmt.Sprintf("%s: %s", entry.WorkspaceID, compactError(err)))
		}
		repoWarnings = appendWarningLines(repoWarnings, entry.WorkspaceID, warnings)
		status, state := loadWorkspaceStatusForRemoval(ctx, rootDir, entry.WorkspaceID)
		items = append(items, workspaceListEntry{entry: entry, repos: repos, state: state, status: status})
	}
	repoWarnings = appendWarningLines(repoWarnings, "", warnings)
	if len(repoWarnings) > 0 {
		renderWarningsSection(renderer, "warnings", repoWarnings, false)
		renderer.Blank()
	}

	renderer.Section("Result")
	var choices []ui.WorkspaceChoice
	for _, item := range items {
		choice := buildWorkspaceChoiceFromRepos(item.entry, item.repos)
		if showDetails {
			choice = buildWorkspaceChoiceWithDetails(ctx, item.entry, item.repos, item.status)
		}
		choice.Warning, choice.WarningStrong = workspaceRemoveWarningLabel(item.state)
		choices = append(choices, choice)
	}
	lines := ui.WorkspaceChoiceLines(choices, -1, useColor, theme)
	if showDetails {
		lines = ui.WorkspaceChoiceConfirmLines(choices, useColor, theme)
	}
	for _, line := range lines {
		renderer.LineRaw(line)
	}
}

func writeRepoListText(entries []repo.Entry, warnings []error) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	warningLines := appendWarningLines(nil, "", warnings)
	if len(warningLines) > 0 {
		renderWarningsSection(renderer, "warnings", warningLines, false)
		renderer.Blank()
	}

	renderer.Section("Result")
	for _, entry := range entries {
		renderer.Bullet(fmt.Sprintf("%s %s", entry.RepoKey, entry.StorePath))
	}
}

func writePresetListText(file preset.File, names []string) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Result")
	if len(names) == 0 {
		renderer.Bullet("no presets found")
		renderSuggestions(renderer, useColor, []string{
			"gwst create --preset",
			"gwst create --preset <name>",
		})
		return
	}
	for _, name := range names {
		renderer.Bullet(name)
		presetEntry, ok := file.Presets[name]
		if !ok {
			continue
		}
		var repos []string
		for _, repoSpec := range presetEntry.Repos {
			display := displayPresetRepo(repoSpec)
			if strings.TrimSpace(display) == "" {
				continue
			}
			repos = append(repos, display)
		}
		renderTreeLines(renderer, repos, treeLineNormal)
	}
	renderSuggestions(renderer, useColor, []string{
		"gwst create --preset",
		"gwst create --preset <name>",
	})
}

func writePresetShowText(name string, presetEntry preset.Preset) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	renderer.Section("Result")
	renderer.Bullet(name)
	if len(presetEntry.Repos) > 0 {
		renderTreeLines(renderer, presetEntry.Repos, treeLineNormal)
	}
}

func writeInitText(result initcmd.Result) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	var skipped []string
	for _, dir := range result.SkippedDirs {
		skipped = append(skipped, fmt.Sprintf("dir: %s", dir))
	}
	for _, file := range result.SkippedFiles {
		skipped = append(skipped, fmt.Sprintf("file: %s", file))
	}
	if len(skipped) > 0 {
		renderer.Section("Info")
		renderer.Bullet("already exists")
		renderTreeLines(renderer, skipped, treeLineNormal)
		renderer.Blank()
	}

	renderer.Section("Steps")
	if len(result.CreatedDirs) == 0 && len(result.CreatedFiles) == 0 {
		renderer.Bullet("no changes")
	} else {
		for _, dir := range result.CreatedDirs {
			renderer.Bullet(fmt.Sprintf("create dir %s", dir))
		}
		for _, file := range result.CreatedFiles {
			renderer.Bullet(fmt.Sprintf("create file %s", file))
		}
	}

	renderer.Blank()
	renderer.Section("Result")
	renderer.Bullet(fmt.Sprintf("root: %s", result.RootDir))

	renderSuggestions(renderer, useColor, []string{
		"gwst preset ls",
		"gwst repo get <repo>",
		fmt.Sprintf("Edit gwst.yaml: %s", filepath.Join(result.RootDir, manifest.FileName)),
	})
}

func writeDoctorText(result doctor.Result, fixed []string) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	if len(result.Warnings) > 0 {
		var lines []string
		for _, warning := range result.Warnings {
			lines = append(lines, warning.Error())
		}
		renderWarningsSection(renderer, "warnings", lines, false)
		renderer.Blank()
	}

	renderer.Section("Result")

	if len(result.Issues) == 0 {
		renderer.Bullet("no issues found")
	} else {
		for _, issue := range result.Issues {
			renderer.BulletError(issue.Kind)
			details := issueDetails(issue)
			renderTreeLines(renderer, details, treeLineError)
		}
	}

	if len(fixed) > 0 {
		renderer.Bullet(fmt.Sprintf("fixed (%d)", len(fixed)))
		var lines []string
		for _, path := range fixed {
			lines = append(lines, path)
		}
		renderTreeLines(renderer, lines, treeLineNormal)
	}

}

func writeDoctorSelfText(result doctor.SelfResult) {
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	if len(result.Warnings) > 0 {
		renderWarningsSection(renderer, "warnings", result.Warnings, false)
		renderer.Blank()
	}

	renderer.Section("Result")
	if len(result.Issues) == 0 {
		renderer.Bullet("no issues found")
	} else {
		for _, issue := range result.Issues {
			renderer.BulletError(issue.Kind)
			details := issueDetails(issue)
			renderTreeLines(renderer, details, treeLineError)
		}
	}

	if len(result.Details) > 0 {
		renderer.Blank()
		renderer.Section("Details")
		renderer.Bullet("environment")
		renderTreeLines(renderer, result.Details, treeLineNormal)
	}
}

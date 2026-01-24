package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/ui"
)

func runManifestRm(ctx context.Context, rootDir string, args []string, globalNoPrompt bool) error {
	rmFlags := flag.NewFlagSet("manifest rm", flag.ContinueOnError)
	var noApply bool
	var noPromptFlag bool
	var helpFlag bool
	rmFlags.BoolVar(&noApply, "no-apply", false, "do not run gion apply")
	rmFlags.BoolVar(&noPromptFlag, "no-prompt", false, "disable interactive prompt")
	rmFlags.BoolVar(&helpFlag, "help", false, "show help")
	rmFlags.BoolVar(&helpFlag, "h", false, "show help")
	rmFlags.SetOutput(os.Stdout)
	rmFlags.Usage = func() {
		printManifestRmHelp(os.Stdout)
	}
	if err := rmFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printManifestRmHelp(os.Stdout)
		return nil
	}

	noPrompt := globalNoPrompt || noPromptFlag

	desired, err := manifest.Load(rootDir)
	if err != nil {
		return err
	}

	manifestPath := manifest.Path(rootDir)
	originalBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", manifest.FileName, err)
	}

	selectedIDs := uniqueNonEmptyStrings(rmFlags.Args())
	selectedFromPrompt := false
	if len(selectedIDs) == 0 {
		if noPrompt {
			return fmt.Errorf("workspace id is required when --no-prompt is set")
		}
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("interactive workspace selection requires a TTY")
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		choices := buildManifestRmWorkspaceChoices(ctx, rootDir, desired)
		selected, err := ui.PromptWorkspaceMultiSelectWithBlocked("gion manifest rm", choices, nil, theme, useColor)
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			return nil
		}
		selectedFromPrompt = true
		selectedIDs = uniqueNonEmptyStrings(selected)
	}
	if len(selectedIDs) == 0 {
		return nil
	}

	var missing []string
	for _, id := range selectedIDs {
		if _, ok := desired.Workspaces[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("workspace(s) not found in %s: %s", manifest.FileName, strings.Join(missing, ", "))
	}

	updated := desired
	if updated.Workspaces == nil {
		updated.Workspaces = map[string]manifest.Workspace{}
	}
	for _, id := range selectedIDs {
		delete(updated.Workspaces, id)
	}

	inputs := func(r *ui.Renderer) {
		r.Section("Inputs")
		for _, id := range selectedIDs {
			desc := ""
			if entry, ok := desired.Workspaces[id]; ok {
				desc = strings.TrimSpace(entry.Description)
			}
			kind := bestEffortWorkspaceRiskKind(ctx, rootDir, id)
			tag := formatManifestRmRiskTag(r, kind)
			line := id + tag
			if desc != "" {
				line += " - " + desc
			}
			r.Bullet(fmt.Sprintf("workspace: %s", line))
		}
	}

	var showPrelude func(*ui.Renderer)
	if !selectedFromPrompt {
		showPrelude = inputs
	}

	return applyManifestMutation(ctx, rootDir, updated, manifestMutationOptions{
		NoApply:       noApply,
		NoPrompt:      noPrompt,
		OriginalBytes: originalBytes,
		Hooks: manifestMutationHooks{
			ShowPrelude: showPrelude,
			RenderNoApply: func(r *ui.Renderer) {
				r.Section("Result")
				r.Bullet(fmt.Sprintf("updated %s (removed %d workspace(s))", manifest.FileName, len(selectedIDs)))
				r.Blank()
				r.Section("Suggestion")
				r.Bullet("gion apply")
			},
			RenderNoChanges: func(r *ui.Renderer) {
				r.Section("Result")
				r.Bullet(fmt.Sprintf("updated %s (removed %d workspace(s))", manifest.FileName, len(selectedIDs)))
				r.Bullet("no changes")
			},
			RenderInfoBeforeApply: func(r *ui.Renderer, _ manifestplan.Result, _ bool) {
				if !selectedFromPrompt {
					r.Section("Info")
				}
				r.Bullet(fmt.Sprintf("manifest: updated %s (removed %d workspace(s))", manifest.FileName, len(selectedIDs)))
				r.Bullet("apply: reconciling entire root (destructive removals require confirmation)")
			},
		},
	})
}

func formatManifestRmRiskTag(r *ui.Renderer, kind workspace.WorkspaceStateKind) string {
	if r == nil {
		return ""
	}
	switch kind {
	case workspace.WorkspaceStateUnknown:
		return r.ErrorText(fmt.Sprintf("[%s]", kind))
	case workspace.WorkspaceStateDirty:
		return r.ErrorText(fmt.Sprintf("[%s]", kind))
	case workspace.WorkspaceStateDiverged, workspace.WorkspaceStateUnpushed:
		return r.WarnText(fmt.Sprintf("[%s]", kind))
	default:
		return ""
	}
}

func buildManifestRmWorkspaceChoices(ctx context.Context, rootDir string, desired manifest.File) []ui.WorkspaceChoice {
	var ids []string
	for id := range desired.Workspaces {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	choices := make([]ui.WorkspaceChoice, 0, len(ids))
	for _, id := range ids {
		ws := desired.Workspaces[id]
		kind := bestEffortWorkspaceRiskKind(ctx, rootDir, id)
		warn, strong := manifestRmRiskWarning(kind)
		var repoChoices []ui.PromptChoice
		for _, repoEntry := range ws.Repos {
			repoName := strings.TrimSpace(repoEntry.Alias)
			if repoName == "" {
				repoName = displayRepoKey(repoEntry.RepoKey)
			}
			label := repoName
			branch := strings.TrimSpace(repoEntry.Branch)
			if branch != "" {
				label = fmt.Sprintf("%s (branch: %s)", repoName, branch)
			}
			var details []string
			repoKey := strings.TrimSpace(repoEntry.RepoKey)
			if repoKey != "" {
				details = append(details, fmt.Sprintf("repo: %s", displayRepoKey(repoKey)))
			}
			repoChoices = append(repoChoices, ui.PromptChoice{
				Label:   label,
				Details: details,
			})
		}
		choices = append(choices, ui.WorkspaceChoice{
			ID:            id,
			Description:   strings.TrimSpace(ws.Description),
			Repos:         repoChoices,
			Warning:       warn,
			WarningStrong: strong,
		})
	}
	return choices
}

func bestEffortWorkspaceRiskKind(ctx context.Context, rootDir, workspaceID string) workspace.WorkspaceStateKind {
	state, err := workspace.State(ctx, rootDir, workspaceID)
	if err != nil {
		return workspace.WorkspaceStateUnknown
	}
	hasDirty := false
	hasUnknown := false
	hasDiverged := false
	hasUnpushed := false
	for _, repo := range state.Repos {
		switch repo.Kind {
		case workspace.RepoStateUnknown:
			hasUnknown = true
		case workspace.RepoStateDirty:
			hasDirty = true
		case workspace.RepoStateDiverged:
			hasDiverged = true
		case workspace.RepoStateUnpushed:
			hasUnpushed = true
		}
	}
	switch {
	case hasUnknown:
		return workspace.WorkspaceStateUnknown
	case hasDirty:
		return workspace.WorkspaceStateDirty
	case hasDiverged:
		return workspace.WorkspaceStateDiverged
	case hasUnpushed:
		return workspace.WorkspaceStateUnpushed
	default:
		return workspace.WorkspaceStateClean
	}
}

func manifestRmRiskWarning(kind workspace.WorkspaceStateKind) (warning string, strong bool) {
	switch kind {
	case workspace.WorkspaceStateUnknown:
		return "unknown", true
	case workspace.WorkspaceStateDirty:
		return "dirty", true
	case workspace.WorkspaceStateDiverged:
		return "diverged", false
	case workspace.WorkspaceStateUnpushed:
		return "unpushed", false
	default:
		return "", false
	}
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

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
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/gitcmd"
	"github.com/tasuku43/gwst/internal/ui"
)

type manifestGcCandidate struct {
	WorkspaceID string
	Targets     []string
	Reason      string
}

func runManifestGc(ctx context.Context, rootDir string, args []string, globalNoPrompt bool) error {
	gcFlags := flag.NewFlagSet("manifest gc", flag.ContinueOnError)
	var noApply bool
	var noPromptFlag bool
	var helpFlag bool
	gcFlags.BoolVar(&noApply, "no-apply", false, "do not run gwiac apply")
	gcFlags.BoolVar(&noPromptFlag, "no-prompt", false, "disable interactive prompt")
	gcFlags.BoolVar(&helpFlag, "help", false, "show help")
	gcFlags.BoolVar(&helpFlag, "h", false, "show help")
	gcFlags.SetOutput(os.Stdout)
	gcFlags.Usage = func() {
		printManifestGcHelp(os.Stdout)
	}
	if err := gcFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printManifestGcHelp(os.Stdout)
		return nil
	}
	if gcFlags.NArg() != 0 {
		return fmt.Errorf("usage: gwiac manifest gc [--no-apply] [--no-prompt]")
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

	ids := make([]string, 0, len(desired.Workspaces))
	for id := range desired.Workspaces {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var warnings []error
	var candidates []manifestGcCandidate
	scanned := 0
	skipped := 0

	for _, id := range ids {
		ws, ok := desired.Workspaces[id]
		if !ok {
			continue
		}
		scanned++

		state, err := workspace.State(ctx, rootDir, id)
		if err != nil {
			warnings = append(warnings, fmt.Errorf("%s: workspace status unavailable: %w", id, err))
			skipped++
			continue
		}
		if state.Kind != workspace.WorkspaceStateClean {
			skipped++
			continue
		}

		var repoTargets []string
		allMerged := true
		for _, repoEntry := range ws.Repos {
			target, ok, err := resolveLocalMergeTarget(ctx, rootDir, repoEntry)
			if err != nil {
				warnings = append(warnings, fmt.Errorf("%s: %s: resolve merge target: %w", id, repoEntryLabel(repoEntry), err))
				allMerged = false
				break
			}
			if !ok {
				warnings = append(warnings, fmt.Errorf("%s: %s: merge target unavailable", id, repoEntryLabel(repoEntry)))
				allMerged = false
				break
			}
			repoTargets = append(repoTargets, fmt.Sprintf("%s=%s", repoEntryLabel(repoEntry), target))

			merged, err := strictMergedIntoTarget(ctx, rootDir, repoEntry, target)
			if err != nil {
				warnings = append(warnings, fmt.Errorf("%s: %s: merged check failed: %w", id, repoEntryLabel(repoEntry), err))
				allMerged = false
				break
			}
			if !merged {
				allMerged = false
				break
			}
		}

		if !allMerged {
			skipped++
			continue
		}
		candidates = append(candidates, manifestGcCandidate{
			WorkspaceID: id,
			Targets:     repoTargets,
			Reason:      "merged",
		})
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	var warningLines []string
	for _, warn := range warnings {
		warningLines = append(warningLines, compactError(warn))
	}
	if len(candidates) == 0 {
		if len(warningLines) > 0 {
			renderer.Section("Info")
			renderManifestGcWarnings(renderer, warningLines)
			renderer.Blank()
		}
		renderer.Section("Result")
		renderer.Bullet("no candidates")
		return nil
	}

	updated := desired
	for _, c := range candidates {
		delete(updated.Workspaces, c.WorkspaceID)
	}

	var candidateIDs []string
	for _, c := range candidates {
		candidateIDs = append(candidateIDs, c.WorkspaceID)
	}

	return applyManifestMutation(ctx, rootDir, updated, manifestMutationOptions{
		NoApply:       noApply,
		NoPrompt:      noPrompt,
		OriginalBytes: originalBytes,
		Hooks: manifestMutationHooks{
			RenderNoApply: func(r *ui.Renderer) {
				r.Section("Info")
				renderManifestGcInfo(r, candidates, warningLines)
				r.Blank()
				r.Section("Result")
				r.Bullet(fmt.Sprintf("updated %s (removed %d workspace(s))", manifest.FileName, len(candidateIDs)))
				r.Blank()
				r.Section("Suggestion")
				r.Bullet("gwiac apply")
			},
			RenderNoChanges: func(r *ui.Renderer) {
				r.Section("Result")
				r.Bullet(fmt.Sprintf("updated %s (removed %d workspace(s))", manifest.FileName, len(candidateIDs)))
				r.Bullet("no changes")
			},
			RenderInfoBeforeApply: func(r *ui.Renderer, plan manifestplan.Result, _ bool) {
				r.Section("Info")
				renderManifestGcInfo(r, candidates, warningLines)
				r.Bullet(r.AccentText("manifest:") + " " + r.SuccessText("updated") + " " + manifest.FileName + " (" + r.ErrorText(fmt.Sprintf("removed %d workspace(s)", len(candidateIDs))) + ")")
				if planIncludesChangesOutsideWorkspaceIDs(plan, candidateIDs) {
					r.Bullet(r.AccentText("apply:") + " reconciling entire root (" + r.WarnText("plan includes changes outside GC scope") + ")")
				} else {
					r.Bullet(r.AccentText("apply:") + " reconciling entire root")
				}
			},
		},
	})
}

func renderManifestGcInfo(r *ui.Renderer, candidates []manifestGcCandidate, warningLines []string) {
	if r == nil {
		return
	}
	if len(warningLines) > 0 {
		renderManifestGcWarnings(r, warningLines)
	}

	if len(candidates) > 0 {
		lines := make([]string, 0, len(candidates))
		for _, c := range candidates {
			reason := strings.TrimSpace(c.Reason)
			if reason == "" {
				reason = "unknown"
			}
			line := fmt.Sprintf("%s %s", c.WorkspaceID, r.MutedText(fmt.Sprintf("[%s]", reason)))
			lines = append(lines, line)
		}
		r.Bullet(fmt.Sprintf("%s %d", r.WarnText("candidates:"), len(candidates)))
		renderTreeLines(r, lines, treeLineNormal)
	}
}

func renderManifestGcWarnings(r *ui.Renderer, warningLines []string) {
	if r == nil || len(warningLines) == 0 {
		return
	}
	r.Bullet(r.WarnText("warnings:"))
	renderTreeLines(r, warningLines, treeLineWarn)
}

func repoEntryLabel(entry manifest.Repo) string {
	alias := strings.TrimSpace(entry.Alias)
	if alias != "" {
		return alias
	}
	return displayRepoKey(entry.RepoKey)
}

func resolveLocalMergeTarget(ctx context.Context, rootDir string, entry manifest.Repo) (string, bool, error) {
	base := strings.TrimSpace(entry.BaseRef)
	if base != "" {
		return base, true, nil
	}

	storePath, exists, err := repo.Exists(rootDir, repo.SpecFromKey(entry.RepoKey))
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, nil
	}

	ref, ok, err := gitcmd.SymbolicRef(ctx, storePath, "refs/remotes/origin/HEAD")
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "refs/remotes/origin/") {
		branch := strings.TrimPrefix(ref, "refs/remotes/origin/")
		if branch != "" {
			return fmt.Sprintf("origin/%s", branch), true, nil
		}
	}
	return "", false, nil
}

func strictMergedIntoTarget(ctx context.Context, rootDir string, entry manifest.Repo, target string) (bool, error) {
	branch := strings.TrimSpace(entry.Branch)
	if branch == "" {
		return false, fmt.Errorf("branch is required")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return false, fmt.Errorf("target is required")
	}

	storePath, exists, err := repo.Exists(rootDir, repo.SpecFromKey(entry.RepoKey))
	if err != nil {
		return false, err
	}
	if !exists {
		return false, fmt.Errorf("repo store not found (run: gwiac repo get %s)", repo.SpecFromKey(entry.RepoKey))
	}

	headRef := fmt.Sprintf("refs/heads/%s", branch)
	targetRef := fmt.Sprintf("refs/remotes/%s", target)

	headHash, headOK, err := gitcmd.ShowRef(ctx, storePath, headRef)
	if err != nil {
		return false, err
	}
	if !headOK {
		return false, fmt.Errorf("ref not found: %s", headRef)
	}
	targetHash, targetOK, err := gitcmd.ShowRef(ctx, storePath, targetRef)
	if err != nil {
		return false, err
	}
	if !targetOK {
		return false, fmt.Errorf("ref not found: %s", targetRef)
	}

	if headHash == targetHash {
		return false, nil
	}

	ok, err := gitcmd.IsAncestor(ctx, storePath, headRef, targetRef)
	if err != nil {
		return false, err
	}
	return ok, nil
}

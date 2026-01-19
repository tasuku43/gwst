package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
	"github.com/tasuku43/gwst/internal/ui"
)

func runPlan(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printPlanHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gwst plan")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	result, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		return err
	}

	var warningLines []string
	for _, warn := range result.Warnings {
		warningLines = append(warningLines, warn.Error())
	}
	if len(warningLines) > 0 {
		renderWarningsSection(renderer, "warnings", warningLines, false)
		renderer.Blank()
	}

	renderer.Section("Result")
	if len(result.Changes) == 0 {
		renderer.Bullet("no changes")
		return nil
	}

	for _, change := range result.Changes {
		switch change.Kind {
		case manifestplan.WorkspaceAdd:
			renderer.BulletSuccess(fmt.Sprintf("+ add workspace %s", change.WorkspaceID))
			renderRepoChanges(renderer, change.Repos)
		case manifestplan.WorkspaceRemove:
			renderer.BulletError(fmt.Sprintf("- remove workspace %s", change.WorkspaceID))
		case manifestplan.WorkspaceUpdate:
			renderer.BulletAccent(fmt.Sprintf("~ update workspace %s", change.WorkspaceID))
			renderRepoChanges(renderer, change.Repos)
		}
	}
	return nil
}

func renderRepoChanges(renderer *ui.Renderer, changes []manifestplan.RepoChange) {
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
			lines = append(lines, formatRepoUpdate(change))
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

func formatRepoUpdate(change manifestplan.RepoChange) string {
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

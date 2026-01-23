package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/manifestls"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/ui"
)

func runManifest(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printManifestHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "ls":
		return runManifestLs(ctx, rootDir, args[1:])
	case "add":
		return runManifestAdd(ctx, rootDir, args[1:], noPrompt)
	case "rm":
		return runManifestRm(ctx, rootDir, args[1:], noPrompt)
	case "validate":
		return runManifestValidate(ctx, rootDir, args[1:])
	case "preset", "pre", "p":
		return runManifestPreset(ctx, rootDir, args[1:], noPrompt)
	default:
		return fmt.Errorf("unknown manifest subcommand: %s", args[0])
	}
}

func runManifestLs(ctx context.Context, rootDir string, args []string) error {
	lsFlags := flag.NewFlagSet("manifest ls", flag.ContinueOnError)
	lsFlags.SetOutput(os.Stdout)
	var helpFlag bool
	var noPrompt bool
	lsFlags.BoolVar(&helpFlag, "help", false, "show help")
	lsFlags.BoolVar(&helpFlag, "h", false, "show help")
	lsFlags.BoolVar(&noPrompt, "no-prompt", false, "disable interactive prompt (no effect)")
	lsFlags.Usage = func() {
		printManifestLsHelp(os.Stdout)
	}
	if err := lsFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = noPrompt
	if helpFlag {
		printManifestLsHelp(os.Stdout)
		return nil
	}
	if lsFlags.NArg() != 0 {
		return fmt.Errorf("usage: gwst manifest ls [--no-prompt]")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	result, err := manifestls.List(ctx, rootDir)
	if err != nil {
		return err
	}

	var warningLines []string
	for _, warn := range result.Warnings {
		warningLines = append(warningLines, compactError(warn))
	}
	if len(warningLines) > 0 {
		renderWarningsSection(renderer, "warnings", warningLines, false)
		renderer.Blank()
	}

	total := result.Counts.Applied + result.Counts.Drift + result.Counts.Missing + result.Counts.Extra
	if total > 0 {
		renderer.Section("Info")
		renderer.Bullet(fmt.Sprintf("applied: %d", result.Counts.Applied))
		renderer.Bullet(fmt.Sprintf("drift: %d", result.Counts.Drift))
		renderer.Bullet(fmt.Sprintf("missing: %d", result.Counts.Missing))
		renderer.Bullet(fmt.Sprintf("extra: %d", result.Counts.Extra))
		renderer.Blank()
	}

	renderer.Section("Result")
	for _, entry := range result.ManifestEntries {
		renderer.Bullet(formatManifestLsLine(renderer, entry.WorkspaceID, entry.Drift, entry.Risk, entry.HasWorkspace, entry.Description, false))
	}
	for _, entry := range result.ExtraEntries {
		renderer.Bullet(formatManifestLsLine(renderer, entry.WorkspaceID, entry.Drift, entry.Risk, entry.HasWorkspace, "", true))
	}
	return nil
}

func formatManifestLsLine(r *ui.Renderer, workspaceID string, drift manifestls.DriftStatus, riskKind workspace.WorkspaceStateKind, hasWorkspace bool, description string, isExtra bool) string {
	line := strings.TrimSpace(workspaceID)
	if line == "" {
		line = "<unknown>"
	}
	if isExtra {
		line += " (extra)"
	} else {
		line += fmt.Sprintf(" (%s)", drift)
	}
	if hasWorkspace {
		tag := strings.TrimSpace(formatRiskTag(r, riskKind))
		if tag != "" {
			line += " " + tag
		}
	}
	desc := strings.TrimSpace(description)
	if !isExtra && desc != "" {
		line += " - " + desc
	}
	return line
}

func formatRiskTag(r *ui.Renderer, kind workspace.WorkspaceStateKind) string {
	if r == nil {
		return ""
	}
	switch kind {
	case workspace.WorkspaceStateUnknown:
		return r.ErrorText(fmt.Sprintf("[%s]", kind))
	case workspace.WorkspaceStateDirty, workspace.WorkspaceStateUnpushed, workspace.WorkspaceStateDiverged:
		return r.WarnText(fmt.Sprintf("[%s]", kind))
	default:
		return ""
	}
}

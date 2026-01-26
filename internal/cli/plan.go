package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gion/internal/app/manifestplan"
	"github.com/tasuku43/gion/internal/domain/manifest"
	"github.com/tasuku43/gion/internal/ui"
)

func runPlan(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printPlanHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gion plan")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	result, err := manifestplan.Plan(ctx, rootDir)
	if err != nil {
		var vErr *manifest.ValidationError
		if errors.As(err, &vErr) {
			renderManifestValidationResult(renderer, vErr.Result)
			return err
		}
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

	renderer.Section("Plan")
	if len(result.Changes) == 0 {
		renderer.Bullet("no changes")
		return nil
	}
	renderPlanChanges(ctx, rootDir, renderer, result)
	return nil
}

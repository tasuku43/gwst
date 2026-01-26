package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gion/internal/domain/manifest"
	"github.com/tasuku43/gion/internal/ui"
)

func runManifestValidate(ctx context.Context, rootDir string, args []string) error {
	validateFlags := flag.NewFlagSet("manifest validate", flag.ContinueOnError)
	validateFlags.SetOutput(os.Stdout)
	var helpFlag bool
	var noPrompt bool
	validateFlags.BoolVar(&helpFlag, "help", false, "show help")
	validateFlags.BoolVar(&helpFlag, "h", false, "show help")
	validateFlags.BoolVar(&noPrompt, "no-prompt", false, "disable interactive prompt (no effect)")
	validateFlags.Usage = func() {
		printManifestValidateHelp(os.Stdout)
	}
	if err := validateFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = noPrompt
	if helpFlag {
		printManifestValidateHelp(os.Stdout)
		return nil
	}
	if validateFlags.NArg() != 0 {
		return fmt.Errorf("usage: gion manifest validate [--no-prompt]")
	}

	result, err := manifest.Validate(ctx, rootDir)
	if err != nil {
		return err
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	renderManifestValidationResult(renderer, result)
	if len(result.Issues) == 0 {
		return nil
	}
	return fmt.Errorf("manifest validation failed")
}

func renderManifestValidationResult(r *ui.Renderer, result manifest.ValidationResult) {
	if r == nil {
		return
	}
	r.Section("Result")
	if len(result.Issues) == 0 {
		r.Bullet("no issues found")
		return
	}
	for _, issue := range result.Issues {
		r.Bullet(formatManifestValidationIssue(issue))
	}
}

func formatManifestValidationIssue(issue manifest.ValidationIssue) string {
	ref := strings.TrimSpace(issue.Ref)
	msg := strings.TrimSpace(issue.Message)
	if ref == "" {
		ref = manifest.FileName
	}
	if msg == "" {
		return ref
	}
	return fmt.Sprintf("%s: %s", ref, msg)
}

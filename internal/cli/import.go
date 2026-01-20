package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/app/manifestimport"
	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/ui"
)

func runImport(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printImportHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gwst import")
	}
	currentFile, err := loadManifestOrEmpty(rootDir)
	if err != nil {
		return err
	}
	currentBytes, err := manifest.Marshal(currentFile)
	if err != nil {
		return err
	}

	nextFile, warnings, err := manifestimport.Build(ctx, rootDir)
	if err != nil {
		return err
	}
	nextBytes, err := manifest.Marshal(nextFile)
	if err != nil {
		return err
	}

	diffLines, err := buildUnifiedDiffLines(currentBytes, nextBytes)
	if err != nil {
		return err
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	result, err := manifestimport.Write(rootDir, nextFile, warnings)
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

	renderer.Section("Diff")
	if len(diffLines) > 0 {
		renderDiffLines(renderer, diffLines)
	} else {
		renderer.Bullet("no changes")
	}
	return nil
}

func loadManifestOrEmpty(rootDir string) (manifest.File, error) {
	file, err := manifest.Load(rootDir)
	if err == nil {
		return file, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return manifest.File{Version: 1, Workspaces: map[string]manifest.Workspace{}}, nil
	}
	return manifest.File{}, err
}

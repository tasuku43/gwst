package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gws/internal/domain/repo"
	"github.com/tasuku43/gws/internal/domain/workspace"
	"github.com/tasuku43/gws/internal/ui"
)

func runPath(rootDir string, args []string, noPrompt bool) error {
	pathFlags := flag.NewFlagSet("path", flag.ContinueOnError)
	var helpFlag bool
	var workspaceFlag bool
	var srcFlag bool
	pathFlags.BoolVar(&workspaceFlag, "workspace", false, "select workspace path")
	pathFlags.BoolVar(&srcFlag, "src", false, "select src path")
	pathFlags.BoolVar(&helpFlag, "help", false, "show help")
	pathFlags.BoolVar(&helpFlag, "h", false, "show help")
	pathFlags.SetOutput(os.Stdout)
	pathFlags.Usage = func() {
		printPathHelp(os.Stdout)
	}
	if err := pathFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printPathHelp(os.Stdout)
		return nil
	}
	if pathFlags.NArg() != 0 {
		return fmt.Errorf("usage: gws path (--workspace | --src)")
	}
	if noPrompt {
		return fmt.Errorf("gws path requires interactive prompt (omit --no-prompt)")
	}
	if workspaceFlag == srcFlag {
		return fmt.Errorf("usage: gws path (--workspace | --src)")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stderr.Fd())

	if workspaceFlag {
		entries, _, err := workspace.List(rootDir)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return fmt.Errorf("no workspaces found")
		}
		choices := make([]ui.PromptChoice, 0, len(entries))
		for _, entry := range entries {
			label := entry.WorkspaceID
			if desc := strings.TrimSpace(entry.Description); desc != "" {
				label = fmt.Sprintf("%s - %s", entry.WorkspaceID, desc)
			}
			choices = append(choices, ui.PromptChoice{
				Label: label,
				Value: entry.WorkspaceID,
			})
		}
		workspaceID, err := ui.PromptChoiceSelectWithOutput("gws path", "workspace", choices, theme, useColor, os.Stderr)
		if err != nil {
			return err
		}
		workspaceID = strings.TrimSpace(workspaceID)
		if workspaceID == "" {
			return fmt.Errorf("workspace id is required")
		}
		fmt.Fprintln(os.Stdout, workspace.WorkspaceDir(rootDir, workspaceID))
		return nil
	}

	srcDirs, _, err := repo.ListSrc(rootDir)
	if err != nil {
		return err
	}
	if len(srcDirs) == 0 {
		return fmt.Errorf("no src directories found")
	}
	choices := make([]ui.PromptChoice, 0, len(srcDirs))
	labelToPath := make(map[string]string, len(srcDirs))
	for _, dir := range srcDirs {
		label := relPath(rootDir, dir)
		if strings.TrimSpace(label) == "" {
			label = filepath.ToSlash(dir)
		}
		labelToPath[label] = dir
		choices = append(choices, ui.PromptChoice{
			Label: label,
			Value: label,
		})
	}
	selectedPath, err := ui.PromptChoiceSelectWithOutput("gws path", "src", choices, theme, useColor, os.Stderr)
	if err != nil {
		return err
	}
	selectedLabel := strings.TrimSpace(selectedPath)
	if selectedLabel == "" {
		return fmt.Errorf("src path is required")
	}
	selectedPath = labelToPath[selectedLabel]
	if strings.TrimSpace(selectedPath) == "" {
		return fmt.Errorf("src path is required")
	}
	fmt.Fprintln(os.Stdout, selectedPath)
	return nil
}

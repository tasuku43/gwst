package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/ui"
)

func runPath(rootDir string, args []string, noPrompt bool) error {
	pathFlags := flag.NewFlagSet("path", flag.ContinueOnError)
	var helpFlag bool
	var workspaceFlag bool
	pathFlags.BoolVar(&workspaceFlag, "workspace", false, "select workspace path")
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
		return fmt.Errorf("usage: gwst path --workspace")
	}
	if noPrompt {
		return fmt.Errorf("gwst path requires interactive prompt (omit --no-prompt)")
	}
	if !workspaceFlag {
		return fmt.Errorf("usage: gwst path --workspace")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stderr.Fd())

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
	workspaceID, err := ui.PromptChoiceSelectWithOutput("gwst path", "workspace", choices, theme, useColor, os.Stderr)
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

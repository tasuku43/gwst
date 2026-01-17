package main

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/cli"
	"github.com/tasuku43/gwst/internal/ui"
)

func main() {
	if err := cli.Run(); err != nil {
		if isatty.IsTerminal(os.Stderr.Fd()) {
			theme := ui.DefaultTheme()
			renderer := ui.NewRenderer(os.Stderr, theme, true)
			renderer.Blank()
			renderer.BulletError(fmt.Sprintf("error: %s", err.Error()))
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

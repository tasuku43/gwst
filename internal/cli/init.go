package cli

import (
	"fmt"
	"os"

	"github.com/tasuku43/gion/internal/app/initcmd"
)

func runInit(rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printInitHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gion init")
	}
	result, err := initcmd.Run(rootDir)
	if err != nil {
		return err
	}
	writeInitText(result)
	return nil
}

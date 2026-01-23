package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/tasuku43/gwst/internal/infra/debuglog"
	"github.com/tasuku43/gwst/internal/infra/paths"
)

// Run is a placeholder for the CLI entrypoint.
func Run() error {
	fs := flag.NewFlagSet("gwst", flag.ContinueOnError)
	var rootFlag string
	var noPrompt bool
	var debugFlag bool
	var helpFlag bool
	var versionFlag bool
	fs.StringVar(&rootFlag, "root", "", "override gwst root")
	fs.BoolVar(&noPrompt, "no-prompt", false, "disable interactive prompt")
	fs.BoolVar(&debugFlag, "debug", false, "write debug logs to file")
	fs.BoolVar(&helpFlag, "help", false, "show help")
	fs.BoolVar(&helpFlag, "h", false, "show help")
	fs.BoolVar(&versionFlag, "version", false, "print version")
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		printGlobalHelp(os.Stdout)
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	args := fs.Args()
	if versionFlag {
		printVersion(os.Stdout)
		return nil
	}
	if helpFlag {
		if len(args) > 0 && printCommandHelp(args[0], os.Stdout) {
			return nil
		}
		printGlobalHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		printGlobalHelp(os.Stdout)
		return nil
	}
	if args[0] == "help" {
		if len(args) > 1 && printCommandHelp(args[1], os.Stdout) {
			return nil
		}
		printGlobalHelp(os.Stdout)
		return nil
	}
	if args[0] == "version" {
		printVersion(os.Stdout)
		return nil
	}

	rootDir, err := paths.ResolveRoot(rootFlag)
	if err != nil {
		return err
	}
	if debugFlag {
		if err := debuglog.Enable(rootDir); err != nil {
			return err
		}
		defer debuglog.Close()
	}

	ctx := context.Background()
	switch args[0] {
	case "init":
		return runInit(rootDir, args[1:])
	case "doctor":
		return runDoctor(ctx, rootDir, args[1:])
	case "repo":
		return runRepo(ctx, rootDir, args[1:])
	case "preset":
		return runPresetRemoved(args[1:])
	case "manifest", "man", "m":
		return runManifest(ctx, rootDir, args[1:], noPrompt)
	case "ls":
		return runLsRemoved(args[1:])
	case "status":
		return runWorkspaceStatus(ctx, rootDir, args[1:])
	case "open":
		return runWorkspaceOpen(ctx, rootDir, args[1:], noPrompt)
	case "path":
		return runPath(rootDir, args[1:], noPrompt)
	case "plan":
		return runPlan(ctx, rootDir, args[1:])
	case "import":
		return runImport(ctx, rootDir, args[1:], noPrompt)
	case "apply":
		return runApply(ctx, rootDir, args[1:], noPrompt)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

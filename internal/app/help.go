package app

import (
	"fmt"
	"io"
	"strings"
)

func isHelpArg(arg string) bool {
	switch strings.TrimSpace(arg) {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func printGlobalHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws <command> [flags] [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  new [--template <name>] [<ID>]     create workspace")
	fmt.Fprintln(w, "  add <ID> <repo>                    add repo to workspace")
	fmt.Fprintln(w, "  ls                                list workspaces")
	fmt.Fprintln(w, "  status <ID>                        workspace status")
	fmt.Fprintln(w, "  rm <ID>                            remove workspace")
	fmt.Fprintln(w, "  review <PR URL>                    create review workspace from PR")
	fmt.Fprintln(w, "  repo <subcommand>                  repo commands (get/ls)")
	fmt.Fprintln(w, "  template <subcommand>              template commands (ls)")
	fmt.Fprintln(w, "  gc [--dry-run] [--older <duration>]")
	fmt.Fprintln(w, "  doctor [--fix]")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  help [command]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Global flags:")
	fmt.Fprintln(w, "  --root <path>      override GWS_ROOT")
	fmt.Fprintln(w, "  --no-prompt        disable interactive prompt")
	fmt.Fprintln(w, "  --json             machine-readable output")
	fmt.Fprintln(w, "  --verbose, -v      verbose logs")
	fmt.Fprintln(w, "  --help, -h         show help")
}

func printCommandHelp(cmd string, w io.Writer) bool {
	switch cmd {
	case "new":
		printNewHelp(w)
	case "add":
		printAddHelp(w)
	case "ls":
		printLsHelp(w)
	case "status":
		printStatusHelp(w)
	case "rm":
		printRmHelp(w)
	case "review":
		printReviewHelp(w)
	case "repo":
		printRepoHelp(w)
	case "template":
		printTemplateHelp(w)
	case "gc":
		printGCHelp(w)
	case "doctor":
		printDoctorHelp(w)
	case "init":
		printInitHelp(w)
	default:
		return false
	}
	return true
}

func printNewHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws new [--template <name>] [<WORKSPACE_ID>]")
	fmt.Fprintln(w, "  --template <name>  template name")
}

func printAddHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws add <WORKSPACE_ID> <repo>")
}

func printLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws ls")
}

func printStatusHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws status <WORKSPACE_ID>")
}

func printRmHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws rm <WORKSPACE_ID>")
}

func printReviewHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws review <PR URL>")
	fmt.Fprintln(w, "  Create a review workspace from a GitHub PR URL")
	fmt.Fprintln(w, "  workspace id: REVIEW-PR-<number>")
	fmt.Fprintln(w, "  forks are not supported / requires gh")
}

func printRepoHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws repo <subcommand>")
	fmt.Fprintln(w, "  subcommands: get, ls")
}

func printRepoGetHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws repo get <repo>")
	fmt.Fprintln(w, "  repo: git@github.com:owner/repo.git | https://github.com/owner/repo.git")
}

func printRepoLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws repo ls")
}

func printTemplateHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template <subcommand>")
	fmt.Fprintln(w, "  subcommands: ls")
}

func printTemplateLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template ls")
}

func printGCHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws gc [--dry-run] [--older <duration>]")
	fmt.Fprintln(w, "  --dry-run          list candidates only")
	fmt.Fprintln(w, "  --older <duration> e.g. 30d, 720h")
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws doctor [--fix]")
}

func printInitHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws init")
}

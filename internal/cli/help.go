package cli

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
	fmt.Fprintln(w, "  new [--template <name>] [<ID>]     create workspace from template")
	fmt.Fprintln(w, "  add [<ID>] [<repo>]                add repo worktree to workspace")
	fmt.Fprintln(w, "  ls                                list workspaces (with repos)")
	fmt.Fprintln(w, "  status [<ID>]                      check dirty/untracked status")
	fmt.Fprintln(w, "  rm [<ID>]                          remove workspace (clean only)")
	fmt.Fprintln(w, "  review <PR URL>                    create review workspace from PR")
	fmt.Fprintln(w, "  repo <subcommand>                  repo commands (get/ls)")
	fmt.Fprintln(w, "  template <subcommand>              template commands (ls)")
	fmt.Fprintln(w, "  doctor [--fix]                     check workspace/repo health")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  help [command]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Global flags:")
	fmt.Fprintln(w, "  --root <path>      override GWS_ROOT")
	fmt.Fprintln(w, "  --no-prompt        disable interactive prompt")
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
	fmt.Fprintln(w, "Usage: gws status [<WORKSPACE_ID>]")
	fmt.Fprintln(w, "  Show dirty/untracked state for each repo")
}

func printRmHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws rm <WORKSPACE_ID>")
}

func printReviewHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws review <PR URL>")
	fmt.Fprintln(w, "  Create a review workspace from a PR/MR URL (GitHub, GitLab, Bitbucket)")
	fmt.Fprintln(w, "  workspace id: REVIEW-PR-<number>")
	fmt.Fprintln(w, "  forks supported; gh not required")
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
	fmt.Fprintln(w, "  subcommands: ls, new")
}

func printTemplateLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template ls")
}

func printTemplateNewHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template new [<name>] [--repo <repo> ...]")
	fmt.Fprintln(w, "  --repo <repo>  repo spec (repeatable)")
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws doctor [--fix]")
}

func printInitHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws init")
}

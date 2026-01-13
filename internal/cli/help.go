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
	fmt.Fprintln(w, "  create [mode flags] [args]         create workspace")
	fmt.Fprintln(w, "  add [<ID>] [<repo>]                add repo worktree to workspace")
	fmt.Fprintln(w, "  ls                                list workspaces (with repos)")
	fmt.Fprintln(w, "  status [<ID>]                      check dirty/untracked status")
	fmt.Fprintln(w, "  rm [<ID>]                          remove workspace (confirms on warnings)")
	fmt.Fprintln(w, "  open [<ID>]                        open workspace in subshell")
	fmt.Fprintln(w, "  path (--workspace | --src)         print selected workspace/src path")
	fmt.Fprintln(w, "  repo <subcommand>                  repo commands (get/ls)")
	fmt.Fprintln(w, "  template <subcommand>              template commands (ls/add/rm)")
	fmt.Fprintln(w, "  doctor [--fix | --self]            check workspace/repo health")
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
	case "create":
		printCreateHelp(w)
	case "add":
		printAddHelp(w)
	case "ls":
		printLsHelp(w)
	case "status":
		printStatusHelp(w)
	case "rm":
		printRmHelp(w)
	case "open":
		printOpenHelp(w)
	case "path":
		printPathHelp(w)
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

func printCreateHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws create [--template <name> | --review [<PR URL>] | --issue [<ISSUE_URL>]] [<WORKSPACE_ID>] [--workspace-id <id>] [--branch <name>] [--base <ref>] [--no-prompt]")
	fmt.Fprintln(w, "  --template <name>  template name")
	fmt.Fprintln(w, "  --review           create review workspace from PR")
	fmt.Fprintln(w, "  --issue            create issue workspace from issue")
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
	fmt.Fprintln(w, "Usage: gws rm [<WORKSPACE_ID>]")
}

func printOpenHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws open [<WORKSPACE_ID>] [--shell]")
	fmt.Fprintln(w, "  Open an interactive subshell at the workspace root")
}

func printPathHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws path (--workspace | --src)")
	fmt.Fprintln(w, "  --workspace       select a workspace path")
	fmt.Fprintln(w, "  --src             select a src path")
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
	fmt.Fprintln(w, "  subcommands: ls, add, rm")
}

func printTemplateLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template ls")
}

func printTemplateAddHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template add [<name>] [--repo <repo> ...]")
	fmt.Fprintln(w, "  --repo <repo>  repo spec (repeatable)")
}

func printTemplateRmHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template rm [<name> ...]")
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws doctor [--fix | --self]")
	fmt.Fprintln(w, "  --self            run self-diagnostics for the gws environment")
}

func printInitHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws init")
}

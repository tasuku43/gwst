package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gws/internal/ui"
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
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gws <command> [flags] [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Commands:"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "init", "initialize gws root layout"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "create [mode flags] [args]", "create workspace (template/review/issue/repo)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "open [<WORKSPACE_ID>] [--shell]", "open workspace in subshell"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "add [<WORKSPACE_ID>] [<repo>]", "add repo worktree to workspace"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "ls", "list workspaces (with repos)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "status [<WORKSPACE_ID>]", "check dirty/untracked status"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "rm [<WORKSPACE_ID>]", "remove workspace (confirms on warnings)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "path --workspace", "print selected workspace path"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "repo <subcommand>", "repo commands (get/ls)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "template <subcommand>", "template commands (ls/add/rm/validate)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "doctor [--fix | --self]", "check workspace/repo health"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "help [command]", "show help for a command"))
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Global flags:"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--root <path>", "override gws root"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "disable interactive prompt"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--debug", "write debug logs to file"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--help, -h", "show help"))
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
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gws create [--template <name> | --review [<PR URL>] | --issue [<ISSUE_URL>] | --repo [<repo>]] [<WORKSPACE_ID>] [--workspace-id <id>] [--branch <name>] [--base <ref>] [--no-prompt]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--template <name>", "template name"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--review [<PR URL>]", "create review workspace from PR"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--issue [<ISSUE_URL>]", "create issue workspace from issue"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--repo [<repo>]", "create workspace from a repo (optional interactive selection)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--workspace-id <id>", "override workspace id (issue mode)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--branch <name>", "override branch name (issue mode)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--base <ref>", "override base ref (issue mode)"))
}

func printAddHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws add [<WORKSPACE_ID>] [<repo>]")
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
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gws path --workspace")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--workspace", "select a workspace path"))
	fmt.Fprintln(w, "  requires interactive prompt (omit --no-prompt)")
}

func printRepoHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gws repo <subcommand>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Subcommands:"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "get <repo>", "fetch or update bare repo store"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "ls", "list known bare repo stores"))
}

func printRepoGetHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gws repo get <repo>")
	fmt.Fprintln(w, helpFlag(theme, useColor, "repo", "git@github.com:owner/repo.git | https://github.com/owner/repo.git"))
}

func printRepoLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws repo ls")
}

func printTemplateHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gws template <subcommand>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Subcommands:"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "ls", "list templates"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "add [<name>]", "add a template"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "rm [<name>]", "remove templates"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "validate", "validate templates.yaml"))
}

func printTemplateLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template ls")
}

func printTemplateAddHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gws template add [<name>] [--repo <repo> ...]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--repo <repo>", "repo spec (repeatable)"))
}

func printTemplateRmHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template rm [<name> ...]")
}

func printTemplateValidateHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws template validate")
	fmt.Fprintln(w, "  Validate templates.yaml")
}

func printDoctorHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gws doctor [--fix | --self]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--fix", "list issues and planned fixes (no changes yet)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--self", "run self-diagnostics for the gws environment"))
}

func printInitHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gws init")
}

func helpTheme(w io.Writer) (ui.Theme, bool) {
	theme := ui.DefaultTheme()
	if file, ok := w.(*os.File); ok {
		return theme, isatty.IsTerminal(file.Fd())
	}
	return theme, false
}

func helpSectionTitle(theme ui.Theme, useColor bool, title string) string {
	if !useColor {
		return title
	}
	return theme.SectionTitle.Render(title)
}

func helpCommand(theme ui.Theme, useColor bool, name, description string) string {
	if useColor {
		return fmt.Sprintf("  %s  %s", theme.Accent.Render(name), description)
	}
	return fmt.Sprintf("  %-30s %s", name, description)
}

func helpFlag(theme ui.Theme, useColor bool, flag, description string) string {
	if useColor {
		return fmt.Sprintf("  %s  %s", theme.Accent.Render(flag), description)
	}
	return fmt.Sprintf("  %-18s %s", flag, description)
}

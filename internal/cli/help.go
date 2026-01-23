package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/ui"
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
	fmt.Fprintln(w, "Usage: gwst <command> [flags] [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Commands:"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "init", "initialize gwst root layout"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "open [<WORKSPACE_ID>] [--shell]", "open workspace in subshell"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "status [<WORKSPACE_ID>]", "check dirty/untracked status"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "path --workspace", "print selected workspace path"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "manifest <subcommand>", "gwst.yaml inventory commands (aliases: man, m)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "repo <subcommand>", "repo commands (get/ls)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "doctor [--fix | --self]", "check workspace/repo health"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "plan", "show gwst.yaml diff (no changes)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "import", "rebuild gwst.yaml from filesystem"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "apply", "apply gwst.yaml to filesystem"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "version", "print gwst version"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "help [command]", "show help for a command"))
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Global flags:"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--root <path>", "override gwst root"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "disable interactive prompt"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--debug", "write debug logs to file"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--version", "print version"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--help, -h", "show help"))
}

func printCommandHelp(cmd string, w io.Writer) bool {
	switch cmd {
	case "ls":
		printLsHelp(w)
	case "status":
		printStatusHelp(w)
	case "open":
		printOpenHelp(w)
	case "path":
		printPathHelp(w)
	case "repo":
		printRepoHelp(w)
	case "preset":
		printPresetHelp(w)
	case "manifest", "man", "m":
		printManifestHelp(w)
	case "doctor":
		printDoctorHelp(w)
	case "plan":
		printPlanHelp(w)
	case "import":
		printImportHelp(w)
	case "apply":
		printApplyHelp(w)
	case "init":
		printInitHelp(w)
	case "version":
		printVersion(w)
	default:
		return false
	}
	return true
}

func printLsHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "gwst ls is removed.")
	fmt.Fprintln(w, "Use: gwst manifest ls")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Legacy usage (no longer supported): gwst ls [--details]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--details", "show git status details (removed)"))
}

func printStatusHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst status [<WORKSPACE_ID>]")
	fmt.Fprintln(w, "  Show dirty/untracked state for each repo")
}

func printOpenHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst open [<WORKSPACE_ID>] [--shell]")
	fmt.Fprintln(w, "  Open an interactive subshell at the workspace root")
}

func printPathHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst path --workspace")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--workspace", "select a workspace path"))
	fmt.Fprintln(w, "  requires interactive prompt (omit --no-prompt)")
}

func printRepoHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst repo <subcommand>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Subcommands:"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "get <repo>", "fetch or update bare repo store"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "ls", "list known bare repo stores"))
}

func printRepoGetHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst repo get <repo>")
	fmt.Fprintln(w, helpFlag(theme, useColor, "repo", "git@github.com:owner/repo.git | https://github.com/owner/repo.git"))
}

func printRepoLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst repo ls")
}

func printPresetHelp(w io.Writer) {
	fmt.Fprintln(w, "gwst preset is removed.")
	fmt.Fprintln(w, "Use: gwst manifest preset")
}

func printManifestHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest <subcommand>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Aliases: gwst man, gwst m")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Subcommands:"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "ls", "list workspace inventory with drift tags"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "add [mode flags] [args]", "add workspace to gwst.yaml then apply (default)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "rm [<WORKSPACE_ID> ...]", "remove workspace entries from gwst.yaml then apply (default)"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "preset <subcommand>", "preset inventory commands (aliases: pre, p)"))
}

func printManifestLsHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest ls [--no-prompt]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "accepted for compatibility (no effect)"))
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Statuses:"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "applied", "manifest and filesystem match (no diff)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "drift", "both exist but differ (would be update in plan/apply)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "missing", "in manifest but missing on filesystem (would be add in plan/apply)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "extra", "on filesystem but missing in manifest (use import to capture)"))
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Tips:"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "gwst plan", "show the full diff details for drift/missing/extra"))
}

func printManifestAddHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest add [--preset <name> | --review [<PR URL>] | --issue <ISSUE_URL> | --repo <repo>] [<WORKSPACE_ID>] [--branch <name>] [--base <ref>] [--no-apply] [--no-prompt]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--preset <name>", "preset name"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--review [<PR URL>]", "add review workspace from PR (GitHub only)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--issue <ISSUE_URL>", "add issue workspace from issue (GitHub only)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--repo <repo>", "add workspace from a repo"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--branch <name>", "override branch name (repo/issue modes only)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--base <ref>", "override base ref (issue mode; applies to all repos in no-prompt)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-apply", "update gwst.yaml only (do not run gwst apply)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "disable interactive prompt"))
}

func printManifestRmHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest rm [<WORKSPACE_ID> ...] [--no-apply] [--no-prompt]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-apply", "update gwst.yaml only (do not run gwst apply)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "disable interactive prompt"))
}

func printManifestPresetHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest preset <subcommand>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Aliases: gwst manifest pre, gwst manifest p")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, helpSectionTitle(theme, useColor, "Subcommands:"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "ls", "list manifest presets"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "add [<name>]", "add a preset entry to gwst.yaml"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "rm [<name> ...]", "remove preset entries from gwst.yaml"))
	fmt.Fprintln(w, helpCommand(theme, useColor, "validate", "validate presets in gwst.yaml"))
}

func printManifestPresetLsHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest preset ls [--no-prompt]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "accepted for compatibility (no effect)"))
}

func printManifestPresetAddHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest preset add [<name>] [--repo <repo> ...] [--no-prompt]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--repo <repo>", "repo spec (repeatable)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "disable interactive prompt"))
}

func printManifestPresetRmHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest preset rm [<name> ...] [--no-prompt]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "disable interactive prompt"))
}

func printManifestPresetValidateHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst manifest preset validate [--no-prompt]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--no-prompt", "accepted for compatibility (no effect)"))
}

func printPresetLsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst preset ls")
}

func printPresetAddHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst preset add [<name>] [--repo <repo> ...]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--repo <repo>", "repo spec (repeatable)"))
}

func printPresetRmHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst preset rm [<name> ...]")
}

func printPresetValidateHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst preset validate")
	fmt.Fprintln(w, "  Validate gwst.yaml")
}

func printDoctorHelp(w io.Writer) {
	theme, useColor := helpTheme(w)
	fmt.Fprintln(w, "Usage: gwst doctor [--fix | --self]")
	fmt.Fprintln(w, helpFlag(theme, useColor, "--fix", "list issues and planned fixes (no changes yet)"))
	fmt.Fprintln(w, helpFlag(theme, useColor, "--self", "run self-diagnostics for the gwst environment"))
}

func printInitHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst init")
}

func printImportHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst import")
}

func printPlanHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst plan")
}

func printApplyHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: gwst apply")
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

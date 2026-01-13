title: "gws ls"
status: implemented
pending:
  - search-filter
  - select-mode
---

## Synopsis
`gws ls [--search <query>] [--select]`

## Intent
List workspaces under `<root>/workspaces` and show a quick view of the repos attached to each.

## Behavior
- Scans `<root>/workspaces` for directories; ignores non-directories.
- For each workspace, scans its contents to discover repo worktrees (alias, repo key, branch, path) and renders them in a tree view.
- If a workspace description is available, show it alongside the workspace ID.
- If a workspace has status warnings (dirty, unpushed, diverged, unknown), show an inline tag next to the workspace ID (same labels as `gws rm`).
- Collects and reports non-fatal warnings from scanning workspaces or repos.
- `--search <query>`: prefilters the list using case-insensitive substring match against workspace ID and repo aliases/paths. Applies to both normal and selection mode.
- `--select`: launches an interactive TUI picker (requires TTY; errors under `--no-prompt`) that lists the filtered workspaces with live search. On `<Enter>`, returns the selected workspace ID to stdout (single line, no sections) and exits 0. If nothing to select, returns an error.

## Success Criteria
- Existing workspaces are listed; command succeeds even if none exist (empty result).

## Failure Modes
- Root path inaccessible or `workspaces/` is not a directory.
- Filesystem or git errors while scanning workspaces (reported as warnings; unrecoverable errors fail the command).
- `--select` used without a TTY, with `--no-prompt`, or when no workspaces match the filter.

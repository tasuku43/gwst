---
title: "gwst open"
status: implemented
---

## Synopsis
`gwst open [<WORKSPACE_ID>] [--shell]`

## Intent
Open a workspace by launching an interactive subshell at the workspace root, making it the primary entrypoint for switching context.

## Behavior
- Accepts optional `WORKSPACE_ID`; if omitted, prompts the user to select a workspace.
- Errors if `GWST_WORKSPACE` is already set (prevents nested `gwst open`).
- Resolves the workspace path as `<root>/workspaces/<WORKSPACE_ID>`.
- Errors if the workspace does not exist.
- Changes the gwst process cwd to the workspace root.
- Spawns the user's default shell in interactive mode (equivalent to `--shell`).
- Uses `$SHELL` if set, otherwise falls back to a sensible default (e.g., `/bin/sh`).
- Wires STDIN/STDOUT/STDERR for direct interaction.
- Optionally sets `GWST_WORKSPACE=<WORKSPACE_ID>` for the child process.
- For shells `bash`, `zsh`, `sh`, prepends a prompt prefix to the child process `PS1`:
  - Prefix format: `[gwst:<WORKSPACE_ID>] ` (blue)
  - If `PS1` is empty or unset, the prefix alone becomes `PS1`.
  - Only the child process receives the modified `PS1`; the parent shell is not changed.
- When the subshell exits, `gwst open` exits and the parent shell cwd remains unchanged.

## Flags
- `--shell`: explicit request to spawn an interactive subshell (default behavior).

## Success Criteria
- `gwst open <WORKSPACE_ID>` starts an interactive shell at the workspace root.
- Exiting the subshell returns the user to the original shell.

## Output
Example:

```
Info
  • subshell; parent cwd unchanged

Steps
  • chdir
    └─ /path/to/gwst/workspaces/OWNER-REPO-ISSUE-19
  • launch subshell
    └─ /bin/zsh -i

Result
  • enter subshell (type `exit` to return)
```

## Failure Modes
- Missing workspace ID.
- `GWST_WORKSPACE` already set (nested `gwst open`).
- Workspace directory not found.
- Unable to determine or start the shell.
- OS-level errors while changing directory or spawning the process.

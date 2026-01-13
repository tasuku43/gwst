---
title: "gws path"
status: implemented
---

## Synopsis
`gws path (--workspace | --src)`

## Intent
Return an absolute path to a selected workspace or src directory for shell usage (e.g., `cd "$(gws path --workspace)"`).

## Behavior
- Requires exactly one of `--workspace` or `--src`.
- `--workspace`:
  - Targets `<root>/workspaces`.
  - Provides a searchable, interactive picker (same UX pattern as `gws create --template` template selection).
  - Search targets: workspace ID and workspace description.
  - Prints the selected workspace path to stdout and nothing else.
- `--src`:
  - Targets `<root>/src`.
  - Provides a searchable, interactive picker (same UX pattern as `gws create --template` template selection).
  - Search targets: directory path only.
  - Prints the selected src directory path to stdout and nothing else.
- Cancel behavior follows existing interactive commands.
- `--no-prompt` is not supported (returns an error).

## Success Criteria
- Stdout contains only the selected path.
- Exit status is 0 on selection.

## Failure Modes
- Both flags specified or neither specified.
- No matching directories found.
- Prompt canceled.
- Filesystem errors while listing directories.

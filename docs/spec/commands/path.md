---
title: "gwst path"
status: implemented
---

## Synopsis
`gwst path --workspace`

## Intent
Return an absolute path to a selected workspace directory for shell usage (e.g., `cd "$(gwst path --workspace)"`).

## Behavior
- Requires `--workspace`.
- Targets `<root>/workspaces`.
- Provides a searchable, interactive picker (same UX pattern as `gwst manifest add --preset` preset selection).
- Search targets: workspace ID and workspace description from `gwst.yaml`.
- Prints the selected workspace path to stdout and nothing else.
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

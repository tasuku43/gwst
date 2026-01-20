---
title: "gwst preset rm"
status: implemented
---

## Synopsis
`gwst preset rm [<name> ...]`

## Intent
Remove one or more preset definitions from `gwst.yaml` without manual editing, with an interactive multi-select when names are not supplied.

## Behavior
- Accepts zero or more preset names. When multiple names are provided, duplicates are removed while preserving first-seen order.
- Requires `gwst.yaml` to exist (`gwst init` completed). Missing file => error.
- With names provided:
  - Fails if any requested name does not exist; no changes are written.
  - Otherwise removes the listed presets and writes the file back via atomic tmp+rename.
- With no names provided and prompts allowed:
  - Opens a filterable list of existing preset names (case-insensitive substring match).
- UI behavior mirrors `gwst preset add` selection: the highlighted item is added on `<Enter>` and removed from the candidate list; a minimum of 1 selection is required.
  - Finish keys: `<Ctrl+D>` or typing `done` then `<Enter>`. If nothing selected, show an error and stay in the loop.
  - With `--no-prompt`, error instead of opening the selector.
- After removal, other presets remain unchanged; repo stores are untouched.
- Output shows the removed preset names under Inputs/Steps/Result; no header line.

## Success Criteria
- `gwst.yaml` is updated atomically and no longer contains the removed presets.

## Failure Modes
- `gwst.yaml` missing or unreadable.
- Preset name not found (when explicitly provided).
- Write/rename failure when persisting the updated file.

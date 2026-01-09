---
title: "gws template rm"
status: planned
---

## Synopsis
`gws template rm [<name> ...]`

## Intent
Remove one or more template definitions from `templates.yaml` without manual editing, with an interactive multi-select when names are not supplied.

## Behavior
- Accepts zero or more template names. When multiple names are provided, duplicates are removed while preserving first-seen order.
- Requires `templates.yaml` to exist (`gws init` completed). Missing file => error.
- With names provided:
  - Fails if any requested name does not exist; no changes are written.
  - Otherwise removes the listed templates and writes the file back via atomic tmp+rename.
- With no names provided and prompts allowed:
  - Opens a filterable list of existing template names (case-insensitive substring match).
  - UI behavior mirrors `gws template new` selection: the highlighted item is added on `<Enter>` and removed from the candidate list; a minimum of 1 selection is required.
  - Finish keys: `<Ctrl+D>` or typing `done` then `<Enter>`. If nothing selected, show an error and stay in the loop.
  - With `--no-prompt`, error instead of opening the selector.
- After removal, other templates remain unchanged; repo stores are untouched.
- Output shows the removed template names under Inputs/Steps/Result; no header line.

## Success Criteria
- `templates.yaml` is updated atomically and no longer contains the removed templates.

## Failure Modes
- `templates.yaml` missing or unreadable.
- Template name not found (when explicitly provided).
- Write/rename failure when persisting the updated file.

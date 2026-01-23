---
title: "gwst manifest preset rm"
status: implemented
aliases:
  - "gwst manifest pre rm"
  - "gwst manifest p rm"
---

## Synopsis
`gwst manifest preset rm [<name> ...] [--no-prompt]`

## Intent
Remove preset entries from `gwst.yaml`.

## Notes
- This command is inventory-only and does not run `gwst apply`.

## Behavior
- Accepts zero or more preset names. When multiple names are provided, duplicates are removed while preserving first-seen order.
- Requires `gwst.yaml` to exist (`gwst init` completed). Missing file => error.
- With names provided:
  - Errors if any requested name does not exist; no changes are written.
  - Otherwise removes the listed presets and writes the file back via atomic tmp+rename.
- With no names provided and prompts allowed:
  - Opens a filterable list of existing preset names (case-insensitive substring match).
  - Multi-select is supported.
  - Cancel/empty selection exits 0 with no changes.
  - With `--no-prompt`, error instead of opening the selector.
- After removal, other presets remain unchanged; repo stores are untouched.
- Output uses the common sectioned layout from `docs/spec/ui/UI.md`. No `Plan`/`Apply` sections are used.

## Interactive selection UX (no args)
- Candidate list is the preset names in `gwst.yaml`.
- Prompt behavior mirrors existing gwst selection UI:
  - Shows a filterable list. Typing narrows candidates by substring match (case-insensitive). Optionally a lightweight fuzzy match is acceptable.
  - The first visible item is highlighted. `<Enter>` adds the highlighted preset name, removes it from the candidate list.
  - The prompt loops, allowing repeated add operations.
  - Finish keys: `<Ctrl+D>` or typing `done` then `<Enter>`.
  - Empty input + `<Enter>` does nothing (stays in the loop) to avoid accidental finish.

## Output examples

### Output: interactive (no args)
```
Inputs
  • preset: he
    └─ helpdesk
    └─ helpers

Result
  • updated gwst.yaml (removed 2 presets)
```

### Output: non-interactive (args)
```
Inputs
  • preset: helpdesk

Result
  • updated gwst.yaml (removed 1 preset)
```

## Success Criteria
- `gwst.yaml` no longer contains the removed preset entries.

## Failure Modes
- `gwst.yaml` missing or unreadable.
- Preset name not found (when explicitly provided).
- Write/rename failure when persisting the updated file.

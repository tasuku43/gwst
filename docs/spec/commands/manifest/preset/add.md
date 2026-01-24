---
title: "gwiac manifest preset add"
status: implemented
aliases:
  - "gwiac manifest pre add"
  - "gwiac manifest p add"
---

## Synopsis
`gwiac manifest preset add [<name>] [--repo <repo> ...] [--no-prompt]`

## Intent
Create a preset entry in `gwiac.yaml` without manual YAML editing.

## Notes
- This command is inventory-only and does not run `gwiac apply`.

## Behavior
- Requires a preset name (`<name>`). Errors if blank or already defined. When omitted and prompts are allowed, asks for the preset name interactively.
- Preset name rules: ASCII letters/digits/`-`/`_`, length 1–64, first char may be digit, case-sensitive. Existing name => error.
- Accepts zero or more `--repo` flags; each must be a valid SSH or HTTPS Git URL. Duplicates are removed while preserving first-seen order.
- When no `--repo` is provided:
  - If prompts are allowed, interactively select repos from the already fetched bare stores (same source as `gwiac repo ls`). Candidates are bare-store repos only (no implicit fetch).
  - Requires at least one selection.
  - With `--no-prompt`, return an error indicating repos are required.
- Validation:
  - `gwiac.yaml` must exist (i.e., `gwiac init` already run); otherwise error.
  - Each repo spec must parse via the existing repospec rules and must already have a bare store fetched; missing stores cause an error (no auto fetch).
- Persistence: load `gwiac.yaml`, add `presets.<name>.repos` using the provided repo strings (trimmed, order preserved), write back via atomic tmp+rename.
- Output: uses the common sectioned layout from `docs/spec/ui/UI.md`. No `Plan`/`Apply` sections are used.

## Interactive selection UX (no --repo)
- Candidate list is the fetched repos from `gwiac repo ls` (already in bare store). Unfetched repos are not shown.
- Prompt behavior mirrors existing gwiac selection UI:
  - Shows a filterable list. Typing narrows candidates by substring match (case-insensitive). Optionally a lightweight fuzzy match is acceptable.
  - The first visible item is highlighted. `<Enter>` adds the highlighted repo, removes it from the candidate list.
  - The prompt loops, allowing repeated add operations. A minimum of 1 selection is required.
  - Finish keys: `<Ctrl+D>` or typing `done` then `<Enter>`. If no repo has been added yet, finishing triggers an error message and returns to the prompt.
  - Empty input + `<Enter>` does nothing (stays in the loop) to avoid accidental finish.

## Output examples

### Output: interactive (no --repo)
```
Inputs
  • preset name: hel
    └─ helmfiles
  • repo: s
    └─ git@github.com:org/repo.git
    └─ git@github.com:org/api.git

Result
  • updated gwiac.yaml
```

### Output: non-interactive (--repo)
```
Inputs
  • preset name: helmfiles
  • repo: git@github.com:org/repo.git
  • repo: git@github.com:org/api.git

Result
  • updated gwiac.yaml
```

## Success Criteria
- `gwiac.yaml` contains a new entry `presets.<name>.repos` with the provided repo specs, and existing presets remain unchanged.

## Failure Modes
- Preset name missing/invalid/already exists.
- No repos supplied and prompting is disabled.
- Repo spec invalid or not fetched (bare store missing).
- `gwiac.yaml` missing/unreadable or cannot be written atomically.

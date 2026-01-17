---
title: "gws template add"
status: implemented
---

## Synopsis
`gws template add <name> [--repo <repo> ...]`

## Intent
Create a template entry in `templates.yaml` without manual YAML editing, so users can quickly define the repo set for new workspaces.

## Behavior
- Requires a template name (`<name>`). Errors if blank or already defined. When omitted and prompts are allowed, asks for the template name interactively.
- Template name rules: ASCII letters/digits/`-`/`_`, length 1–64, first char may be digit, case-sensitive. Existing name => error.
- Accepts zero or more `--repo` flags; each must be a valid SSH or HTTPS Git URL. Duplicates are removed while preserving first-seen order.
- When no `--repo` is provided:
  - If prompts are allowed, interactively select repos from the already fetched repo stores (`gws repo ls` results). User can pick multiple one-by-one and finish with “done”. Requires at least one selection.
  - With `--no-prompt`, return an error indicating repos are required.
- Validation:
  - `templates.yaml` must exist (i.e., `gws init` already run); otherwise error.
  - Each repo spec must parse via the existing repospec rules and must already have a bare store fetched; missing stores cause an error (no auto fetch).
- Persistence: load `templates.yaml`, add `templates.<name>.repos` using the provided repo strings (trimmed, order preserved), write back via atomic tmp+rename.
- Output: prints the created template name and repo list. No JSON output.

## Interactive selection UX (no --repo)
- Candidate list is the fetched repos from `gws repo ls` (already in bare store). Unfetched repos are not shown.
- Prompt behavior (matches existing gws selection UI):
  - Shows a filterable list. Typing narrows candidates by substring match (case-insensitive). Optionally a lightweight fuzzy match is acceptable.
  - The first visible item is highlighted. `<Enter>` adds the highlighted repo, removes it from the candidate list, and echoes `+ added: <repo>`.
  - The prompt loops, allowing repeated add operations. A minimum of 1 selection is required.
  - Finish keys: `<Ctrl+D>` or typing `done` then `<Enter>`. If no repo has been added yet, finishing triggers an error message and returns to the prompt.
  - Empty input + `<Enter>` does nothing (stays in the loop) to avoid accidental finish.
- After selection completes, the command writes `templates.yaml` and renders the standard section blocks consistent with other gws commands (no header line; section “Result” lists the template name and repos).

## Success Criteria
- `templates.yaml` contains a new entry `templates.<name>.repos` with the provided repo specs, and existing templates remain unchanged.

## Failure Modes
- Template name missing/invalid/already exists.
- No repos supplied and prompting is disabled.
- Repo spec invalid or not fetched (bare store missing).
- `templates.yaml` missing/unreadable or cannot be written atomically.

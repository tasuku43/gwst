---
title: "gwiac manifest ls"
status: implemented
aliases:
  - "gwiac man ls"
  - "gwiac m ls"
migrated_from: "docs/spec/commands/manifest-ls.md"
---

## Synopsis
`gwiac manifest ls [--root <path>] [--no-prompt]`

## Intent
List the workspace inventory in `gwiac.yaml` (desired state) and show a lightweight per-workspace drift indicator by scanning the filesystem (actual state).

This is the primary "what do I have and is it applied?" command.

## Behavior
- Loads `<root>/gwiac.yaml`; errors if missing or invalid.
- Scans `<root>/workspaces` to build the current filesystem state.
- For each workspace in the manifest, computes a status summary:
  - `applied`: no diff.
  - `missing`: present in manifest, missing on filesystem (would be `add` in plan/apply).
  - `drift`: present in both but differs (would be `update` in plan/apply).
- Optionally (best-effort), computes a lightweight workspace risk tag by scanning attached repo worktrees:
  - Uses the same labels as the workspace picker: `dirty`, `unpushed`, `diverged`, `unknown` (clean is omitted).
  - Semantics and detection follow the `gwiac manifest rm` "Workspace State Model" (no implicit fetch; do not warn for behind-only).
  - Workspace risk tag is an aggregation of repo risks using the priority defined in `docs/spec/ui/UI.md` (unknown > dirty > diverged > unpushed).
  - Risk tags are shown only when the workspace exists on the filesystem.
- Also detects filesystem-only workspaces (present on filesystem, missing in manifest) and reports them as `extra`.
  - `extra` entries are informational only; use `gwiac import` to capture them into the manifest, or `gwiac apply` (with confirmation) to remove them.
- `extra` entries are included in `Result` after the manifest entries so users can see the full picture of "what exists under this root".
- No changes are made (read-only).
- `--no-prompt` is accepted but has no effect (kept for CLI consistency).

## Output
Uses the common sectioned layout. No interactive UI is required.

- `Info` (optional):
  - counts for `applied`, `missing`, `drift`, `extra`
  - optional per-workspace notes (e.g. `extra: <id>`, scan warnings)
- `Result`:
  - workspace list sorted by workspace id
  - each workspace line includes:
    - `<WORKSPACE_ID>`
    - drift status in parentheses: `(applied|drift|missing)`
    - optional risk tag in brackets when non-clean: `[dirty|unpushed|diverged|unknown]`
    - optional description suffix: ` - <description>`
  - extra entries are appended after the manifest list:
    - sorted by workspace id
    - shown as `<WORKSPACE_ID> (extra)` with an optional risk tag

Example:
```
Info
  • applied: 3
  • drift: 1
  • missing: 1
  • extra: 1

Result
  • PROJ-123 (applied) - fix login flow
  • PROJ-124 (drift) [dirty] - wip refactor
  • PROJ-125 (missing) - onboarding
  • PROJ-OLD (extra) [unknown]
```

## Success Criteria
- Inventory workspaces are listed and drift is accurately classified.

## Failure Modes
- Manifest missing or invalid.
- Filesystem or git errors while scanning workspaces (reported as warnings where possible).

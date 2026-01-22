---
title: "gwst manifest rm"
status: planned
aliases:
  - "gwst man rm"
  - "gwst m rm"
pending:
  - interactive-selection-from-rm
  - risk-context-and-output
---

## Synopsis
`gwst manifest rm [<WORKSPACE_ID> ...] [--no-apply] [--no-prompt]`

## Intent
Remove workspace entries from the inventory (`gwst.yaml`) using an interactive UX (same intent as the legacy `gwst rm` selection), then reconcile the filesystem via `gwst apply` by default.

## Behavior (high level)
- Targets one or more workspaces:
  - With args: treat as the selected workspace IDs.
  - Without args: interactive multi-select (same UX as legacy `gwst rm`).
- Workspace IDs are treated as workspace directory identifiers (they do not have to be valid git branch names).
- Updates `<root>/gwst.yaml` by removing the selected workspace entries.
- By default, runs `gwst apply` to reconcile the filesystem with the updated manifest.
  - Destructive behavior is enforced by `gwst apply` (and `--no-prompt` must error if removals exist).
- With `--no-apply`, stops after rewriting `gwst.yaml` and prints a suggestion to run `gwst apply` next.

## Detailed flow (conceptual)
1. Determine targets (args vs interactive selection).
2. Validate targets:
   - Each selected ID must exist in `gwst.yaml` (inventory-driven removal).
   - If any selected ID is missing from `gwst.yaml`, the command errors and makes no changes.
3. Rewrite `gwst.yaml` (full-file rewrite) to remove the selected workspace entries.
4. If `--no-apply` is set: stop after manifest rewrite.
5. Otherwise run `gwst apply` for the entire root:
   - This may include unrelated drift in the same root.
   - Destructive confirmation rules are handled by `gwst apply`.
6. If apply is cancelled at the confirmation step (`n`/No or `Ctrl-C`), restore the previous `gwst.yaml` from a backup/snapshot.

## Output (IA)
- Always uses the common sectioned layout from `docs/spec/ui/UI.md`.
- `Inputs`: selection inputs (workspace ids, short status tags, optional description).
- `Plan`/`Apply`/`Result`: delegated to `gwst apply` when apply is run.

### Info section (when apply runs)
When apply runs, `gwst manifest rm` should emit an `Info` section after `Inputs` to make the two-phase behavior explicit:
- `manifest: updated gwst.yaml (removed N workspace(s))`
- `apply: reconciling entire root (destructive removals require confirmation)`

## Risk context (guidance)
This command should preserve the spirit of the legacy `gwst rm` UX:
- In interactive selection, show warning indicators for risky workspaces (dirty/unpushed/diverged/unknown).
- Keep the selection UI lightweight: show only a short aggregated status tag next to each workspace (e.g. `[clean]`, `[dirty]`, `[unpushed]`, `[diverged]`, `[unknown]`).
  - Repo-level details are optional; plan output follows and is the primary place for deep review.
- Detailed risk output should primarily come from the `gwst apply` plan output (same format as `gwst plan`), so users can review before confirming destructive removals.

Example (interactive selection, conceptual):
```
Inputs
  • workspace: s
    └─ PROJ-123[clean] - fix login flow
```

## Success Criteria
- Selected workspace entries are removed from `gwst.yaml`.
- When apply is run and confirmed, filesystem no longer contains removed workspaces.

## Failure Modes
- Workspace selection empty/canceled (interactive).
- Any selected workspace ID is missing from `gwst.yaml` (no changes are made).
- Manifest write failure.
- `gwst apply` failure (git/filesystem).

---
title: "gwst manifest rm"
status: planned
aliases:
  - "gwst man rm"
  - "gwst m rm"
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
- When `--no-prompt` is set:
  - If no `<WORKSPACE_ID>` args are provided, error (cannot enter interactive selection).

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

### Inputs section (interactive selection)
When no `<WORKSPACE_ID>` args are provided, the command enters an interactive multi-select picker.

Picker guidance:
- Search/filter is supported (filter text is shown as the input value in `Inputs`).
- Multi-select is supported.
- Cancel/empty selection exits 0 with no changes.

Example (interactive selection, conceptual):
```
Inputs
  • workspace: s
    └─ PROJ-123 - fix login flow
    └─ PROJ-999[dirty] - wip refactor
```

### Info section (when apply runs)
When apply runs, `gwst manifest rm` should emit an `Info` section after `Inputs` to make the two-phase behavior explicit:
- `manifest: updated gwst.yaml (removed N workspace(s))`
- `apply: reconciling entire root (destructive removals require confirmation)`

## Risk context (guidance)
This command should preserve the spirit of the legacy `gwst rm` UX:
- In interactive selection, show warning indicators for risky workspaces (dirty/unpushed/diverged/unknown).
- Keep the selection UI lightweight: show only a short aggregated status tag next to each workspace when non-clean (e.g. `[dirty]`, `[unpushed]`, `[diverged]`, `[unknown]`); omit the tag for clean.
  - Repo-level details are optional; plan output follows and is the primary place for deep review.
- Detailed risk output should primarily come from the `gwst apply` plan output (same format as `gwst plan`), so users can review before confirming destructive removals.
- Status aggregation priority (if multiple conditions apply): `unknown` > `dirty` > `diverged` > `unpushed` (clean is omitted).

Example (interactive selection, conceptual):
```
Inputs
  • workspace: s
    └─ PROJ-123 - fix login flow
```

## Output examples

### Output: `--no-apply`
When `--no-apply` is set, `gwst manifest rm` does not run apply and prints a short summary.

Example:
```
Inputs
  • workspace: PROJ-123

Result
  • updated gwst.yaml (removed 1 workspace)

Suggestion
  gwst apply
```

### Output: with apply (default)
When apply runs, `gwst manifest rm` prints `Inputs` first, then streams `gwst apply` output (`Info`/`Plan`/`Apply`/`Result`).
`gwst manifest rm` itself does not attempt to summarize the plan beyond what `gwst apply` prints.

## Success Criteria
- Selected workspace entries are removed from `gwst.yaml`.
- When apply is run and confirmed, filesystem no longer contains removed workspaces.

## Error messages (guidance)
`gwst manifest rm` should keep errors actionable and include the next command when possible.

Common cases:
- `--no-prompt` with no `<WORKSPACE_ID>` args: error and suggest providing args or removing `--no-prompt`.
- Any selected workspace ID is missing from `gwst.yaml`: error and include the missing ids (no changes are made).
- Apply fails after manifest rewrite:
  - Treat as apply failure and keep the manifest change (users can re-run `gwst apply`).

## Failure Modes
- Workspace selection empty/canceled (interactive): exit 0 (no changes).
- Any selected workspace ID is missing from `gwst.yaml` (no changes are made).
- Manifest write failure.
- `gwst apply` failure (git/filesystem).

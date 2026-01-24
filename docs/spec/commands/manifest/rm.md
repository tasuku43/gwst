---
title: "gion manifest rm"
status: planned
aliases:
  - "gion man rm"
  - "gion m rm"
---

## Synopsis
`gion manifest rm [<WORKSPACE_ID> ...] [--no-apply] [--no-prompt]`

## Intent
Remove workspace entries from the inventory (`gion.yaml`) using an interactive UX, then reconcile the filesystem via `gion apply` by default.

## Behavior (high level)
- Targets one or more workspaces:
  - With args: treat as the selected workspace IDs.
  - Without args: interactive multi-select.
- Workspace IDs are treated as workspace directory identifiers (they do not have to be valid git branch names).
- Updates `<root>/gion.yaml` by removing the selected workspace entries.
- By default, runs `gion apply` to reconcile the filesystem with the updated manifest.
  - Destructive behavior is enforced by `gion apply` (and `--no-prompt` must error if removals exist).
- With `--no-apply`, stops after rewriting `gion.yaml` and prints a suggestion to run `gion apply` next.
- When `--no-prompt` is set:
  - If no `<WORKSPACE_ID>` args are provided, error (cannot enter interactive selection).

## Detailed flow (conceptual)
1. Determine targets (args vs interactive selection).
2. Validate targets:
   - Each selected ID must exist in `gion.yaml` (inventory-driven removal).
   - If any selected ID is missing from `gion.yaml`, the command errors and makes no changes.
3. Rewrite `gion.yaml` (full-file rewrite) to remove the selected workspace entries.
4. If `--no-apply` is set: stop after manifest rewrite.
5. Otherwise run `gion apply` for the entire root:
   - This may include unrelated drift in the same root.
   - Destructive confirmation rules are handled by `gion apply`.
6. If apply is cancelled at the confirmation step (`n`/No or `Ctrl-C`), restore the previous `gion.yaml` from a backup/snapshot.

## Output (IA)
- Always uses the common sectioned layout from `docs/spec/ui/UI.md`.
- `Inputs`: selection inputs (workspace ids, short status tags, optional description).
- `Plan`/`Apply`/`Result`: delegated to `gion apply` when apply is run.

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
When apply runs, `gion manifest rm` should emit an `Info` section after `Inputs` to make the two-phase behavior explicit:
- `manifest: updated gion.yaml (removed N workspace(s))`
- `apply: reconciling entire root (destructive removals require confirmation)`

## Risk context (guidance)
This command should provide lightweight risk context before apply runs:
- In interactive selection, show warning indicators for risky workspaces (dirty/unpushed/diverged/unknown).
- Keep the selection UI lightweight: show only a short aggregated status tag next to each workspace when non-clean (e.g. `[dirty]`, `[unpushed]`, `[diverged]`, `[unknown]`); omit the tag for clean.
  - Repo-level details are optional; plan output follows and is the primary place for deep review.
- Detailed risk output should primarily come from the `gion apply` plan output (same format as `gion plan`), so users can review before confirming destructive removals.
- Status aggregation priority (if multiple conditions apply): `unknown` > `dirty` > `diverged` > `unpushed` (clean is omitted).

### Workspace State Model (picker tags)
The picker status tags (`dirty`/`unpushed`/`diverged`/`unknown`) follow these semantics and detection rules.

Definitions (per repo, based on local state only):
- **Clean**: no uncommitted changes; upstream set; not ahead/behind.
- **Dirty**: uncommitted changes exist (including unmerged/conflicts).
- **Unpushed**: local branch is ahead of upstream.
- **Diverged**: local branch is both ahead and behind upstream.
- **Unknown**: status cannot be determined or branch/upstream cannot be resolved (e.g. upstream missing, detached HEAD).

Detection guidance:
- Source of truth: `git status --porcelain=v2 -b` (local remote-tracking refs; no implicit fetch/prune).
- Do **not** warn for behind-only (upstream advanced with no local changes).

Example (interactive selection, conceptual):
```
Inputs
  • workspace: s
    └─ PROJ-123 - fix login flow
```

## Output examples

### Output: `--no-apply`
When `--no-apply` is set, `gion manifest rm` does not run apply and prints a short summary.

Example:
```
Inputs
  • workspace: PROJ-123

Result
  • updated gion.yaml (removed 1 workspace)

Suggestion
  gion apply
```

### Output: with apply (default)
When apply runs, `gion manifest rm` prints `Inputs` first, then streams `gion apply` output (`Info`/`Plan`/`Apply`/`Result`).
`gion manifest rm` itself does not attempt to summarize the plan beyond what `gion apply` prints.

## Success Criteria
- Selected workspace entries are removed from `gion.yaml`.
- When apply is run and confirmed, filesystem no longer contains removed workspaces.

## Error messages (guidance)
`gion manifest rm` should keep errors actionable and include the next command when possible.

Common cases:
- `--no-prompt` with no `<WORKSPACE_ID>` args: error and suggest providing args or removing `--no-prompt`.
- Any selected workspace ID is missing from `gion.yaml`: error and include the missing ids (no changes are made).
- Apply fails after manifest rewrite:
  - Treat as apply failure and keep the manifest change (users can re-run `gion apply`).

## Failure Modes
- Workspace selection empty/canceled (interactive): exit 0 (no changes).
- Any selected workspace ID is missing from `gion.yaml` (no changes are made).
- Manifest write failure.
- `gion apply` failure (git/filesystem).

---
title: "gwst apply"
status: planned
---

## Synopsis
`gwst apply [--root <path>] [--no-prompt]`

## Intent
Reconcile the filesystem to match `gwst.yaml` by computing a diff, showing a plan, and applying the changes after confirmation.

## Behavior
- Loads `<root>/gwst.yaml`; errors if missing or invalid.
- Scans `<root>/workspaces` to build the current state.
- Computes a plan with `add`, `remove`, and `update` actions:
  - `add`: workspace or repo entry exists in manifest but not on filesystem.
  - `remove`: exists on filesystem but not in manifest.
  - `update`: exists in both but differs by repo alias, repo key, or branch.
- Renders a human-readable plan summary before any changes (same format as `gwst plan`).
- By default, prompts for confirmation if any changes exist.
  - `remove` actions are marked as destructive.
  - If only non-destructive adds are present, prompt can be skipped with `--no-prompt`.
- If confirmed, applies actions in a stable order: removes, then updates, then adds.
- Updates `gwst.yaml` by rewriting the full file after successful apply.

## Flags
- `--no-prompt`: skip confirmation (errors if any removals are present).

## Success Criteria
- Filesystem state matches the manifest.
- `gwst.yaml` is rewritten to a normalized form.

## Failure Modes
- Manifest file missing or invalid.
- Filesystem or git errors while applying actions.
- `--no-prompt` used with destructive actions.

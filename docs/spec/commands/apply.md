---
title: "gwiac apply"
status: implemented
---

## Synopsis
`gwiac apply [--root <path>] [--no-prompt]`

## Intent
Reconcile the filesystem to match `gwiac.yaml` by computing a diff, showing a plan, and applying the changes after confirmation.

## Behavior
- Loads `<root>/gwiac.yaml`; errors if missing or invalid.
- Scans `<root>/workspaces` to build the current state.
- Computes a plan with `add`, `remove`, and `update` actions:
  - `add`: workspace or repo entry exists in manifest but not on filesystem.
  - `remove`: exists on filesystem but not in manifest.
  - `update`: exists in both but differs by repo alias, repo key, or branch.
- Renders a human-readable plan summary before any changes (same format as `gwiac plan`).
- By default, prompts for confirmation if any changes exist.
  - `remove` actions are marked as destructive.
  - If only non-destructive adds are present, prompt can be skipped with `--no-prompt`.
  - For destructive actions, the prompt does not repeat per-repo git status output; users should review the plan output above before confirming.
- If confirmed, applies actions in a stable order: removes, then updates, then adds.
  - When a repo update is a branch rename only (same repo key, different branch), gwiac renames the branch in-place (no worktree remove/add) to match common local development workflows.
- When applying `add` actions that require creating a new branch:
  - If the target `branch` already exists in the bare store, gwiac checks it out when adding the worktree.
  - If the branch does not exist, gwiac creates it from:
    - `base_ref` if present in the repo entry in `gwiac.yaml`, otherwise
    - the repo's detected default branch (prefer `refs/remotes/origin/HEAD`), otherwise fallback heuristics (`HEAD`, then common branch names).
- When gwiac creates a new branch during apply, it records the chosen base as `base_branch` in the workspace `.gwiac/metadata.json` (workspace-level, optional) so a future `gwiac import` can restore `base_ref` in `gwiac.yaml`.
- Updates `gwiac.yaml` by rewriting the full file after successful apply.

## Output (IA)
- `Plan` section: plan summary (same as `gwiac plan`).
  - When interactive, the final confirmation prompt is rendered at the end of `Plan` (with a blank line before it).
- `Apply` section: execution steps, with partial git command logs nested under each step.
- `Result` section: completion summary (e.g. applied counts) and manifest rewrite note.

## Flags
- `--no-prompt`: skip confirmation (errors if any removals are present).

## Success Criteria
- Filesystem state matches the manifest.
- `gwiac.yaml` is rewritten to a normalized form.

## Failure Modes
- Manifest file missing or invalid.
- Filesystem or git errors while applying actions.
- `--no-prompt` used with destructive actions.

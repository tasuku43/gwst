---
title: "IaC Backlog"
status: planned
since: "2026-01-21"
---

# IaC Backlog (manifest-first)

This backlog tracks IaC-style workflow work for gion:

- Users primarily edit `gion.yaml` (via `gion manifest ...`).
- Reconciliation happens only via `gion apply` (which includes plan + confirmation).
- `gion plan` is read-only diff.
- `gion import` rebuilds inventory from filesystem.

## Guiding rules (non-negotiables)
- Interactive UX is preserved for creation/removal flows.
- Mutations update `gion.yaml` first; filesystem is reconciled only by `gion apply`.
- Destructive actions require explicit confirmation; `--no-prompt` must not allow destructive changes.
- Idempotent apply: repeated runs converge with no changes.

## Next TODOs (working set)

Policy:
- Keep this branch focused on specs/docs as much as possible.
- Do implementation work in a separate PR/branch and track it here explicitly.
- Keep the command surface manifest-first (inventory via `gion manifest ...`, execution via `gion apply`).
- Prefer reusing existing interactive input UX within `gion manifest ...` commands.

### Specs / Docs (this branch)
- [x] Finalize `gion manifest add` details (no-prompt requirements, multi-select UX, error messages, output IA).
- [x] Decide `--base` scope for multi-repo (preset): per-repo in interactive flows (prefill via `--base`), workspace-wide only in `--no-prompt`.
- [x] Confirm base tracking model: workspace-level `.gion/metadata.json base_branch` (and `gion.yaml repos[].base_ref`).
- [x] Decide rollback behavior when apply confirmation is declined: restore a backup of `gion.yaml` (do not use import).
- [x] Specify `gion manifest rm` UX and how risk context is shown (in rm vs rely on apply plan).
- [x] Specify `gion manifest ls` drift badges (applied/missing/drift/extra), sorting, and output IA.
- [x] Specify `gion manifest preset` subcommands (ls/add/rm/validate).
- [x] Align `gion import` spec with current implementation.

### Implementation (separate PR)
- [x] Implement CLI routing + aliases (`manifest`/`man`/`m`, `manifest preset`/`pre`/`p`). (spec: `docs/spec/commands/manifest/README.md`)
- [x] Implement `gion manifest ls`. (spec: `docs/spec/commands/manifest/ls.md`, UI: `docs/spec/ui/UI.md`)
- [x] Implement `gion manifest add`. (spec: `docs/spec/commands/manifest/add.md`, UI: `docs/spec/ui/UI.md`)
- [x] Implement `gion manifest rm`. (spec: `docs/spec/commands/manifest/rm.md`, UI: `docs/spec/ui/UI.md`)
- [x] Implement `gion manifest preset` subcommands. (specs: `docs/spec/commands/manifest/preset/*.md`)
- [x] Add manifest validation surface (`gion manifest validate`).
  - Validate `gion.yaml` schema + invariants (workspace ids, repo keys, alias uniqueness, branch/base_ref formats, etc.)
  - Output: actionable errors (non-zero exit), suitable for humans and CI
- [x] Make `gion plan` fail on manifest validation errors (non-zero exit; do not print a diff/plan when invalid).

## Command map (high level)

Inventory (desired state):
- `gion manifest` (aliases: `man`, `m`)
  - Default target: workspaces (`gion manifest add/rm/ls`)
  - Presets: `gion manifest preset ...` (aliases: `pre`, `p`)

Reconcile / drift:
- `gion plan` (full diff, read-only)
- `gion apply` (plan + prompt + reconcile)
- `gion import` (filesystem -> manifest)

## Backlog

### 1) Specs (contracts)
- Add command specs:
  - `docs/spec/commands/manifest/README.md`
  - `docs/spec/commands/manifest/ls.md`
  - New: `docs/spec/commands/manifest/add.md`
  - New: `docs/spec/commands/manifest/rm.md`
  - New: `docs/spec/commands/manifest/preset/*.md` (ls/add/rm/validate)
- Update existing specs to reflect current state.
- Update UI spec examples where new commands affect section layout or prompts.

### 2) CLI surface (routing, help, aliases)
- Add `manifest` command router with aliases `man` and `m`.
- Add `manifest preset` router with aliases `pre` and `p`.
- Ensure help lists alias forms minimally (avoid clutter) but keeps discoverability.

Spec references:
- `docs/spec/commands/manifest/README.md`

### 3) Manifest editing primitives (library layer)
- Implement a manifest read/modify/write package:
  - Read `<root>/gion.yaml` (schema v1)
  - Write normalized `gion.yaml` (full rewrite)
  - Operations:
    - Add/update workspace entry (mode/description/source_url/preset_name)
    - Add/update repo entries (alias/repo_key/branch)
    - Optional base tracking (`base_ref` in `gion.yaml`, `base_branch` in `.gion/metadata.json`)
    - Remove workspace entries
    - Preset add/rm/validate operations
- Validation:
  - Workspace IDs and branch names must pass `git check-ref-format --branch`.
  - Repo key format must match store keys.
  - Alias uniqueness within a workspace.

Spec references:
- Core model: `docs/spec/core/INVENTORY.md`, `docs/spec/core/METADATA.md`
- UI output: `docs/spec/ui/UI.md`

### 4) `gion manifest ls` (inventory list + drift badges)
- Implement per-workspace summary classification:
  - `applied`, `missing`, `drift`, plus filesystem-only `extra`.
- Keep it lightweight; full details remain in `gion plan`.
- Output must follow `docs/spec/ui/UI.md` section order.

Spec references:
- `docs/spec/commands/manifest/ls.md`
- `docs/spec/ui/UI.md`

### 5) `gion manifest add` (interactive inventory authoring)
- Preserve the interactive selection UX:
  - Modes: preset / repo / review / issue (as today)
  - Inputs section remains a single in-place interaction
- Replace the final "filesystem create" step with:
  - Update `gion.yaml` to add the workspace definition
  - By default run `gion apply` (show plan + confirm + reconcile)
  - With `--no-apply`, stop after manifest rewrite and suggest `gion apply` / `gion plan`
- GitHub PR URL handling:
  - Use `gh` for PR/issue metadata fetch (same as current create spec)

Spec references:
- `docs/spec/commands/manifest/add.md`
- `docs/spec/ui/UI.md`

### 6) `gion manifest rm` (replace rm flow)
- Preserve the interactive selection UX from `gion rm` (multi-select, warnings).
- Replace destructive filesystem removal with:
  - Update `gion.yaml` to remove the workspace entry(ies)
  - Run `gion apply` (which handles destructive confirmation rules)
- Ensure the UX still surfaces risk context (lightweight tags in selection; deep review in apply plan).

Spec references:
- `docs/spec/commands/manifest/rm.md`
- `docs/spec/ui/UI.md`

### 7) Preset commands under `gion manifest`
- Implement:
  - `gion manifest preset add`: create preset entries without manual YAML editing
  - `gion manifest preset rm`: remove preset entries
  - `gion manifest preset ls`: list presets
  - `gion manifest preset validate`: validate manifest presets

Spec references:
- `docs/spec/commands/manifest/preset/add.md`
- `docs/spec/commands/manifest/preset/rm.md`
- `docs/spec/commands/manifest/preset/ls.md`
- `docs/spec/commands/manifest/preset/validate.md`

### 8) Apply / plan / import alignment
- Ensure `gion apply` remains the only executor:
  - Any manifest mutations that imply destructive actions must still require prompt through apply.
- Confirm drift semantics:
  - How to classify workspace-level drift vs repo-level drift for list output.
- `gion import` is implemented; keep aligning behavior/output as needed.

Spec references:
- `docs/spec/commands/apply.md`
- `docs/spec/commands/plan.md`
- `docs/spec/commands/import.md`

### 9) Docs & guides
- Update `docs/guides/USECASES.md` with new workflows:
  - "Add workspace to manifest then apply"
  - "Remove workspace from manifest then apply"
  - "List inventory and check drift"
- Add onboarding notes:
  - Recommended workflows (`gion.yaml` + `gion plan`/`gion apply`/`gion import`)

### 10) Tests & validation
- Add/adjust tests for:
  - Manifest edit operations (pure functions where possible)
  - CLI routing and alias behavior
  - `gion manifest ls` classification
- Always run: `gofmt -w .`, `go test ./...`, `go vet ./...`, `go build ./...`

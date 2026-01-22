---
title: "IaC Migration Backlog"
status: planned
since: "2026-01-21"
---

# IaC Migration Backlog (manifest-first)

This backlog tracks the work to shift gwst from "direct filesystem commands" to an IaC-style workflow:

- Users primarily edit `gwst.yaml` (via `gwst manifest ...`).
- Reconciliation happens only via `gwst apply` (which includes plan + confirmation).
- `gwst plan` is read-only diff.
- `gwst import` rebuilds inventory from filesystem.

## Guiding rules (non-negotiables)
- Interactive UX is preserved for creation/removal flows.
- Mutations update `gwst.yaml` first; filesystem is reconciled only by `gwst apply`.
- Destructive actions require explicit confirmation; `--no-prompt` must not allow destructive changes.
- Idempotent apply: repeated runs converge with no changes.

## Next TODOs (working set)

Policy:
- Keep this branch focused on specs/docs as much as possible.
- Do implementation work in a separate PR/branch and track it here explicitly.

### Specs / Docs (this branch)
- [x] Finalize `gwst manifest add` details (no-prompt requirements, multi-select UX, error messages, output IA).
- [x] Decide `--base` scope for multi-repo (preset): per-repo in interactive flows (prefill via `--base`), workspace-wide only in `--no-prompt`.
- [x] Confirm base tracking model: workspace-level `.gwst/metadata.json base_branch` (and `gwst.yaml repos[].base_ref`).
- [ ] Specify `gwst manifest rm` UX and how risk context is shown (in rm vs rely on apply plan).
- [ ] Specify `gwst manifest ls` drift badges (applied/missing/drift/extra), sorting, and output IA.
- [ ] Decide removal behavior for legacy `gwst preset *` (remove vs temporary alias vs hard error).
- [ ] Decide removal behavior for `gwst ls` (exact error message guidance).

### Implementation (separate PR)
- [ ] Implement CLI routing + aliases (`manifest`/`man`/`m`, `manifest preset`/`pre`/`p`).
- [ ] Remove `gwst ls` command (hard error + suggestion to use `gwst manifest ls`).
- [ ] Implement `gwst manifest ls`.
- [ ] Implement `gwst manifest add`.
- [ ] Implement `gwst manifest rm`.
- [ ] Implement `gwst manifest preset` subcommands.

## Command migration map (high level)

Inventory (desired state):
- New: `gwst manifest` (aliases: `man`, `m`)
  - Default target: workspaces (`gwst manifest add/rm/ls`)
  - Presets: `gwst manifest preset ...` (aliases: `pre`, `p`)

Reconcile / drift:
- Keep: `gwst plan` (full diff, read-only)
- Keep: `gwst apply` (plan + prompt + reconcile)
- Keep: `gwst import` (filesystem -> manifest)

Removed / replaced:
- Remove: `gwst ls` (use `gwst manifest ls`)
- Legacy (to be replaced by manifest flows): `create`, `add`, `rm`, `preset *`

## Backlog

### 1) Specs (contracts)
- Add command specs:
  - `docs/spec/commands/manifest/README.md`
  - `docs/spec/commands/manifest/ls.md`
  - New: `docs/spec/commands/manifest/add.md`
  - New: `docs/spec/commands/manifest/rm.md`
  - New: `docs/spec/commands/manifest/preset/*.md` (ls/add/rm/validate)
- Update existing specs to reflect migration state:
  - Mark replaced commands as `legacy` and point to `superseded_by`.
  - Decide on (and document) removal behavior for `gwst ls` (error message guidance).
- Update UI spec examples where new commands affect section layout or prompts.

### 2) CLI surface (routing, help, aliases)
- Add `manifest` command router with aliases `man` and `m`.
- Add `manifest preset` router with aliases `pre` and `p`.
- Remove `ls` command:
  - `gwst ls` should error and suggest `gwst manifest ls`.
  - Remove from global help and command help.
- Ensure help lists alias forms minimally (avoid clutter) but keeps discoverability.

### 3) Manifest editing primitives (library layer)
- Implement a manifest read/modify/write package:
  - Read `<root>/gwst.yaml` (schema v1)
  - Write normalized `gwst.yaml` (full rewrite)
  - Operations:
    - Add/update workspace entry (mode/description/source_url/preset_name)
    - Add/update repo entries (alias/repo_key/branch)
    - Optional base tracking (`base_ref` in `gwst.yaml`, `base_branch` in `.gwst/metadata.json`)
    - Remove workspace entries
    - Preset add/rm/validate operations
- Validation:
  - Workspace IDs and branch names must pass `git check-ref-format --branch`.
  - Repo key format must match store keys.
  - Alias uniqueness within a workspace.

### 4) `gwst manifest ls` (inventory list + drift badges)
- Implement per-workspace summary classification:
  - `applied`, `missing`, `drift`, plus filesystem-only `extra`.
- Keep it lightweight; full details remain in `gwst plan`.
- Output must follow `docs/spec/ui/UI.md` section order.

### 5) `gwst manifest add` (replace create flows)
- Preserve the interactive selection UX from `gwst create`:
  - Modes: preset / repo / review / issue (as today)
  - Inputs section remains a single in-place interaction
- Replace the final "filesystem create" step with:
  - Update `gwst.yaml` to add the workspace definition
  - By default run `gwst apply` (show plan + confirm + reconcile)
  - With `--no-apply`, stop after manifest rewrite and suggest `gwst apply` / `gwst plan`
- GitHub PR URL handling:
  - Use `gh` for PR/issue metadata fetch (same as current create spec)

### 6) `gwst manifest rm` (replace rm flow)
- Preserve the interactive selection UX from `gwst rm` (multi-select, warnings).
- Replace destructive filesystem removal with:
  - Update `gwst.yaml` to remove the workspace entry(ies)
  - Run `gwst apply` (which handles destructive confirmation rules)
- Ensure the UX still surfaces risk context:
  - Either show summarized risk in manifest rm itself before apply
  - Or rely on `gwst apply` plan output to show risk for removals (decision needed)

### 7) Preset commands under `gwst manifest`
- Implement:
  - `gwst manifest preset add`: create preset entries without manual YAML editing
  - `gwst manifest preset rm`: remove preset entries
  - `gwst manifest preset ls`: list presets
  - `gwst manifest preset validate`: validate manifest presets
- Decide if legacy `gwst preset ...` remains as alias temporarily or is removed immediately.

### 8) Apply / plan / import alignment
- Ensure `gwst apply` remains the only executor:
  - Any manifest mutations that imply destructive actions must still require prompt through apply.
- Confirm drift semantics:
  - How to classify workspace-level drift vs repo-level drift for list output.
- Implement `gwst import` (currently planned) early if needed to support real-world drift capture.

### 9) Docs & guides
- Update `docs/guides/USECASES.md` with new workflows:
  - "Add workspace to manifest then apply"
  - "Remove workspace from manifest then apply"
  - "List inventory and check drift"
- Add migration notes:
  - Which commands are removed / replaced
  - Recommended replacements

### 10) Tests & validation
- Add/adjust tests for:
  - Manifest edit operations (pure functions where possible)
  - CLI routing and alias behavior
  - `gwst manifest ls` classification
  - Removal behavior for `gwst ls`
- Always run: `gofmt -w .`, `go test ./...`, `go vet ./...`, `go build ./...`

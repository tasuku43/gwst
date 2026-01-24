---
title: "gion import"
status: implemented
pending:
  - stable-manifest-ordering
  - align-output-with-common-layout
---

## Synopsis
`gion import [--root <path>] [--no-prompt]`

## Intent
Rebuild `gion.yaml` from the filesystem and `.gion/metadata.json` to restore the canonical workspace inventory.

## Behavior
- Scans `<root>/workspaces` to build the current filesystem state.
- For each workspace:
  - Loads `.gion/metadata.json` when present to restore optional metadata fields (`mode`, `description`, `preset_name`, `source_url`, `base_branch`).
  - Derives repo branches from each worktree's Git state.
- If `base_branch` is present in metadata, import should store it as `base_ref` in `gion.yaml` repo entries for the workspace (used only when creating missing branches in future apply runs).
- Presets are preserved from the existing manifest (best-effort): if `<root>/gion.yaml` exists and is readable, `presets` are copied into the imported manifest.
- Workspaces are scanned in sorted order by workspace id; repos are written in sorted order by repo alias.
- Rewrites `<root>/gion.yaml` as a whole, reflecting the current filesystem state.
  - Current implementation overwrites the file directly (no confirmation prompt).
- `--no-prompt` is accepted but currently has no effect (kept for CLI consistency).

## Output
- Reports warnings for unreadable workspaces or invalid metadata under a `warnings` section.
- Prints a unified diff under a `Diff` section (or `no changes` when identical).
  - The diff is computed between the current manifest bytes (or an empty manifest if missing) and the imported manifest bytes.

## Success Criteria
- `gion.yaml` reflects the current filesystem state.

## Failure Modes
- Root directory missing or inaccessible.
- Filesystem errors while scanning workspaces.
- Invalid metadata that prevents import (reported as warnings; fatal only if no valid workspaces remain).
- Failure to write `<root>/gion.yaml`.

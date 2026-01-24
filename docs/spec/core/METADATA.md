---
title: "workspace metadata.json"
status: planned
---

# workspace metadata.json

Each workspace stores minimal metadata under its `.gion` directory so information can be restored when importing from the filesystem.

## Location

`<GION_ROOT>/workspaces/<WORKSPACE_ID>/.gion/metadata.json`

## Source of truth

- During normal commands, gion writes both `gion.yaml` and `.gion/metadata.json`.
- During import/rebuild from filesystem, gion reads `.gion/metadata.json` to restore metadata fields.
- Repo branch names are derived from each worktree's Git state when importing (not stored in metadata).

## Format

```json
{
  "description": "fix login flow",
  "mode": "issue",
  "preset_name": "backend",
  "source_url": "https://github.com/org/repo/issues/123",
  "base_branch": "origin/main"
}
```

## Fields

- `description` (optional): workspace description.
- `mode` (optional): one of `preset`, `repo`, `review`, `issue`, `resume`, `add`.
- `preset_name` (optional): set only when `mode=preset`.
- `source_url` (optional): set when created from a URL (issue/review) or other modes with known origin.
- `base_branch` (optional): base branch/ref used when creating new branches for this workspace the first time.
  - When omitted, the implicit default is the repo's detected default branch (typically `refs/remotes/origin/HEAD`).
  - When present, import may use it to restore `base_ref` in `gion.yaml` so future re-creation can cut from the same base.

## Validation rules

- If `mode` is present, it must be one of the supported values.
- `preset_name` is required when `mode=preset`.
- `source_url` must be a valid URL when present.
- `base_branch` is optional.
  - When present, it must be in the form `origin/<branch>`.
  - `<branch>` must be a non-empty string (no whitespace).

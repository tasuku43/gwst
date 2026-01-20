---
title: "workspace metadata.json"
status: planned
---

# workspace metadata.json

Each workspace stores minimal metadata under its `.gwst` directory so information can be restored when importing from the filesystem.

## Location

`<GWST_ROOT>/workspaces/<WORKSPACE_ID>/.gwst/metadata.json`

## Source of truth

- During normal commands, gwst writes both `gwst.yaml` and `.gwst/metadata.json`.
- During import/rebuild from filesystem, gwst reads `.gwst/metadata.json` to restore metadata fields.
- Repo branch names are derived from each worktree's Git state when importing (not stored in metadata).

## Format

```json
{
  "description": "fix login flow",
  "mode": "issue",
  "preset_name": "backend",
  "source_url": "https://github.com/org/repo/issues/123"
}
```

## Fields

- `description` (optional): workspace description.
- `mode` (required): one of `preset`, `repo`, `review`, `issue`, `resume`, `add`.
- `preset_name` (optional): set only when `mode=preset`.
- `source_url` (optional): set when created from a URL (issue/review) or other modes with known origin.

## Validation rules

- `mode` must be one of the supported values.
- `preset_name` is required when `mode=preset`.
- `source_url` must be a valid URL when present.

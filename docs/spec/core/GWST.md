---
title: "gwst.yaml"
status: planned
---

# gwst.yaml

`gwst.yaml` is the centralized inventory of workspaces used for tracking creation mode and enabling future IaC-style reconciliation.

## Location

`<GWST_ROOT>/gwst.yaml`

Created by:

```
gwst init
```

## Source of truth

- **Normal commands** (`create`, `add`, `rm`, `resume`, etc.): filesystem operations are the truth. After a successful change, `gwst` rewrites `gwst.yaml` as a whole to reflect the new state.
- **`gwst apply`**: `gwst.yaml` is the truth. `gwst` computes a diff, shows the plan, and applies the changes to the filesystem after confirmation.
- **`gwst import`**: filesystem and `.gwst/metadata.json` are the truth. `gwst` rebuilds `gwst.yaml` from the current state.

Notes:
- `gwst.yaml` is a gwst-managed file. Commands rewrite the full file; comments and ordering may not be preserved.
- When rewriting, gwst preserves existing metadata for untouched workspaces where possible, and may read `.gwst/metadata.json` to refill fields like `mode`, `description`, `preset_name`, and `source_url` during imports.
- Repo branch names are derived from each worktree's Git state when importing from the filesystem.

## Format

Top-level keys:
- `version` (required): integer schema version. Initial version is `1`.
- `workspaces` (required): map keyed by workspace ID.
- `presets` (optional): map keyed by preset name.

Workspace entry fields:
- `description` (optional): string.
- `mode` (required): one of `preset`, `repo`, `review`, `issue`, `resume`, `add`.
- `preset_name` (optional): preset name when `mode=preset`.
- `source_url` (optional): source URL for `issue`/`review` (or other modes if available).
- `repos` (required): array of repo entries.

Repo entry fields:
- `alias` (required): directory name under the workspace.
- `repo_key` (required): repo store key, e.g. `github.com/org/repo.git`.
- `branch` (required): branch checked out in the worktree.

```yaml
version: 1
presets:
  webapp:
    repos:
      - git@github.com:org/api.git
      - git@github.com:org/web.git
workspaces:
  PROJ-123:
    description: "fix login flow"
    mode: "issue"
    repos:
      - alias: api
        repo_key: github.com/org/api.git
        branch: issue/123
      - alias: web
        repo_key: github.com/org/web.git
        branch: PROJ-123
```

## Validation rules
- Workspace IDs must be valid git branch names (`git check-ref-format --branch`).
- `mode` must be one of the supported values.
- `repo_key` must match the bare store key format (`<host>/<owner>/<repo>.git`).
- `alias` must be unique within a workspace.
- `branch` must be a valid git branch name.

## Diff semantics (for apply)

When reconciling, gwst computes a plan with three categories:
- **add**: present in gwst.yaml, missing on filesystem.
- **remove**: present on filesystem, missing in gwst.yaml.
- **update**: present in both but differing repo/branch/alias definitions.

Removals are treated as destructive and require explicit confirmation.

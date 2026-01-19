---
title: "manifest.yaml"
status: planned
---

# manifest.yaml

`manifest.yaml` is the centralized inventory of workspaces used for tracking creation mode and enabling future IaC-style reconciliation.

## Location

`<GWST_ROOT>/manifest.yaml`

Created by:

```
gwst init
```

## Source of truth

- **Normal commands** (`create`, `add`, `rm`, `resume`, etc.): filesystem operations are the truth. After a successful change, `gwst` rewrites `manifest.yaml` as a whole to reflect the new state.
- **`gwst apply`**: `manifest.yaml` is the truth. `gwst` computes a diff, shows the plan, and applies the changes to the filesystem after confirmation.
- **`gwst import`**: filesystem and `.gwst/metadata.json` are the truth. `gwst` rebuilds `manifest.yaml` from the current state.

Notes:
- `manifest.yaml` is a gwst-managed file. Commands rewrite the full file; comments and ordering may not be preserved.
- When rewriting, gwst preserves existing metadata for untouched workspaces where possible, and may read `.gwst/metadata.json` to refill fields like `mode`, `description`, `template_name`, and `source_url` during imports.
- Repo branch names are derived from each worktree's Git state when importing from the filesystem.

## Format

Top-level keys:
- `version` (required): integer schema version. Initial version is `1`.
- `workspaces` (required): map keyed by workspace ID.

Workspace entry fields:
- `description` (optional): string.
- `mode` (required): one of `template`, `repo`, `review`, `issue`, `resume`, `add`.
- `template_name` (optional): template name when `mode=template`.
- `source_url` (optional): source URL for `issue`/`review` (or other modes if available).
- `repos` (required): array of repo entries.

Repo entry fields:
- `alias` (required): directory name under the workspace.
- `repo_key` (required): repo store key, e.g. `github.com/org/repo.git`.
- `branch` (required): branch checked out in the worktree.

```yaml
version: 1
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
- **add**: present in manifest, missing on filesystem.
- **remove**: present on filesystem, missing in manifest.
- **update**: present in both but differing repo/branch/alias definitions.

Removals are treated as destructive and require explicit confirmation.

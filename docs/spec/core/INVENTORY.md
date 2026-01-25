---
title: "gion.yaml"
status: implemented
---

# gion.yaml

`gion.yaml` is the centralized inventory of workspaces used for tracking creation mode and enabling future IaC-style reconciliation.

## Location

`<GION_ROOT>/gion.yaml`

Created by:

```
gion init
```

## Source of truth

- **Non-IaC commands**: filesystem operations are the truth. After a successful change, `gion` rewrites `gion.yaml` as a whole to reflect the new state.
- **`gion apply`**: `gion.yaml` is the truth. `gion` computes a diff, shows the plan, and applies the changes to the filesystem after confirmation.
- **`gion import`**: filesystem and `.gion/metadata.json` are the truth. `gion` rebuilds `gion.yaml` from the current state.

Notes:
- `gion.yaml` is a gion-managed file. Commands rewrite the full file; comments and ordering may not be preserved.
- When rewriting, gion preserves existing metadata for untouched workspaces where possible, and may read `.gion/metadata.json` to refill fields like `mode`, `description`, `preset_name`, and `source_url` during imports.
- When importing, gion may also read `.gion/metadata.json` `base_branch` and store it as `base_ref` in `gion.yaml` (per repo entry) to preserve how branches were originally cut.
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
- `base_ref` (optional): base ref used when creating the branch for the first time (only relevant if the branch does not already exist in the store).
  - When present, it must be in the form `origin/<branch>`.
  - If omitted, gion uses the repo's detected default branch (prefers `refs/remotes/origin/HEAD`).

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
        base_ref: origin/main
      - alias: web
        repo_key: github.com/org/web.git
        branch: PROJ-123
```

## Validation rules
- Workspace IDs must satisfy git branch ref format rules (`git check-ref-format --branch`) and must not include path separators or path traversal (`/`, `\\`, `.`, `..`).
- `mode` must be one of the supported values.
- `repo_key` must match the bare store key format (`<host>/<owner>/<repo>.git`) or the normalized repo key form (`<host>/<owner>/<repo>`).
- `alias` must be unique within a workspace.
- `branch` must be a valid git branch name.
- `base_ref` is optional. When provided, it must resolve in the repo store when it is needed to create a new branch (otherwise apply fails).
  - Additionally, `base_ref` must be in the form `origin/<branch>`.

## Diff semantics (for apply)

When reconciling, gion computes a plan with three categories:
- **add**: present in gion.yaml, missing on filesystem.
- **remove**: present on filesystem, missing in gion.yaml.
- **update**: present in both but differing repo/branch/alias definitions.

Removals are treated as destructive and require explicit confirmation.

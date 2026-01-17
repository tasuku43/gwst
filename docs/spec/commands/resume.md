---
title: "gws resume"
status: planned
---

## Synopsis
`gws resume <repo> <ref> [--workspace <id>] [--branch <name>] [--reuse] [--no-prompt]`

## Intent
Rehydrate work from a remote branch or tag by fetching the ref into the bare store and attaching it to a workspace as a worktree, without disturbing existing worktrees.

## Behavior
- Inputs:
  - `<repo>`: SSH/HTTPS repo URL (same format as `gws repo get`).
  - `<ref>`: remote branch or tag name. If it looks like a full ref (`refs/heads/...`, `refs/tags/...`, `origin/...`), it is used as-is; otherwise treated as a branch or tag on `origin`.
- Workspace selection:
  - If `--workspace` is given, target that workspace (must exist).
  - Otherwise, prompt to select a workspace (like `gws add`). If none exist, error.
- Prompt supports filtering (like `gws add`): type-to-filter, Enter to select, selection removes the item, Ctrl+D/done to finish.
- Fetch:
  - Run `git fetch origin <ref>` in the bare store (creating the store via `gws repo get` if missing; prompts unless `--no-prompt`, where it errors instead).
  - If the remote ref is already up to date (same object id locally), skip fetch.
- Worktree creation:
  - Branch name for the worktree defaults to `<ref>` when it is a branch on origin; if `<ref>` resolves to a tag, a branch name is required via `--branch <name>` (no detached checkout by default to avoid accidental edits on a tag).
  - If `--branch` is not provided and prompting is allowed, offer an interactive picker over remote branches/tags (filterable like other prompts) with the initial highlight on `<ref>` if it exists.
  - Destination: `<root>/workspaces/<workspace_id>/<alias>` where alias defaults to repo name.
  - If a worktree for the target branch already exists in the workspace:
    - With `--reuse`, do not create a new worktree; instead, print its path and suggest `cd` (no checkout/reset performed).
    - Without `--reuse`, error.
- Base ref: when the branch doesn’t exist locally, create it from `origin/<ref>` (for branches) or from the fetched tag commit when `--branch` is specified for tags.
  - If the remote branch/tag does not exist, error (advise to use `gws create --template` for new branches).
- Output: standard gws style (no header line; steps, result, suggestion). Result lists workspace, alias, branch, and path.

## Success Criteria
- The bare store has fetched `<ref>`, and the target workspace has a worktree checked out to the chosen branch (newly created or reused with `--reuse`).

## Failure Modes
- Repo spec invalid or store missing and `--no-prompt` forbids `repo get`.
- Workspace not found (or none to select).
- Tag specified without `--branch` for a writable branch name.
- Worktree for the branch already present and `--reuse` not set.
- Git fetch/worktree errors.

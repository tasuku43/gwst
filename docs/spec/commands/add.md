---
title: "gwst add"
status: implemented
---

## Synopsis
`gwst add [<WORKSPACE_ID>] [<repo>]`

## Intent
Attach another repo worktree to an existing workspace, using the workspace ID as the branch name.

## Behavior
- Accepts optional `WORKSPACE_ID` and `repo`; if either is missing, interactively prompts the user to choose a workspace and a repo store discovered via `gwst repo ls`. Fails if no candidates exist.
- Validates that the workspace exists and the ID is a valid git branch name.
- Normalizes the repo spec and defaults the alias to the repo name. Errors if the alias or repo key already exists in the workspace.
- Opens the bare store without fetching (expects `gwst repo get` to have been run already).
- Detects the base ref (prefers `HEAD`, then `origin/HEAD`, then `main`/`master`/`develop` locally or on origin).
- Adds a worktree at `<root>/workspaces/<WORKSPACE_ID>/<alias>`:
  - If branch `<WORKSPACE_ID>` exists in the store, checks it out.
  - Otherwise creates the branch from the detected base ref and checks it out.

## Success Criteria
- New worktree directory exists under the workspace, checked out to branch `<WORKSPACE_ID>`.

## Failure Modes
- Workspace not found or ID invalid.
- Repo store missing (user must run `gwst repo get` first).
- Alias or repo already present in the workspace.
- Base ref cannot be detected.
- Git errors while creating the worktree.

---
title: "gws rm"
status: implemented
---

## Synopsis
`gws rm [<WORKSPACE_ID>]`

## Intent
Safely remove a workspace and all of its worktrees, refusing when repositories are dirty or status cannot be read.

## Behavior
- With `WORKSPACE_ID` provided: targets that workspace.
- Without it: scans workspaces, classifies each as removable or blocked (dirty or status errors), and prompts the user to choose removable entries using the same add/remove loop as `gws create` issue Step 3. Fails if none are removable.
- Multi-select UX:
  - Shows only removable entries for selection.
  - Blocked entries are listed in an Info section (same as current behavior) but are not selectable.
  - `<Enter>` adds the highlighted workspace to the selection list and removes it from candidates.
  - Finish keys: `<Ctrl+D>` or typing `done` then `<Enter>`; minimum 1 selection required.
  - Empty input + `<Enter>` does nothing (stays in loop).
  - Filterable list by substring (case-insensitive); lightweight fuzzy match is acceptable.
- Before removal, gathers warnings (e.g., ahead-of-upstream, missing upstream, status errors) and displays them.
- Before removal, if multiple workspaces are selected, ask for confirmation:
  - Prompt: `Remove N workspaces? (y/N)`
  - Default is No.
- Calls `workspace.Remove`, which:
  - Validates the workspace exists.
  - Fails if any repo has uncommitted/untracked/unstaged/unmerged changes.
  - Runs `git worktree remove <worktree>` for each repo’s worktree.
  - Deletes the workspace directory.

## Success Criteria
- Workspace directory no longer exists; associated worktrees are removed from their bare stores.

## Failure Modes
- Workspace not found.
- Dirty worktrees or status errors block removal.
- Git errors while removing worktrees.
- Filesystem errors while deleting the workspace directory.

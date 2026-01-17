---
title: "gwst rm"
status: implemented
---

## Synopsis
`gwst rm [<WORKSPACE_ID>]`

## Intent
Safely remove a workspace and all of its worktrees, warning/confirming for risky states.

## Workspace State Model
- **Clean**: no uncommitted changes; upstream set; no ahead/behind.
- **Dirty**: uncommitted changes exist (confirmation required).
- **Unpushed**: local branch is ahead of upstream (confirmation required).
- **Diverged**: local branch is both ahead and behind upstream (confirmation required).
- **Unknown**: status cannot be determined or branch/upstream cannot be resolved (confirmation required).

## Warning/Confirmation Rules
- Source of truth for comparison: `git status --porcelain=v2 -b` (local remote-tracking refs; no implicit fetch/prune).
- Warn/confirm when any repo is:
  - Dirty or unmerged.
  - Ahead of upstream (unpushed).
  - Ahead and behind upstream (diverged).
  - Upstream missing or status cannot be determined (unknown).
  - Detached HEAD / no HEAD (unknown).
- Do **not** warn for behind-only (upstream advanced with no local changes).
- Workspace-level warning is the aggregation of repo warnings; Dirty/Unknown are strong warnings.

## Behavior
- With `WORKSPACE_ID` provided: targets that workspace.
- Without it: scans workspaces and prompts the user to choose entries using the same add/remove loop as `gwst create` issue Step 3. Fails if none exist.
- Multi-select UX:
  - Shows entries for selection (including any saved workspace descriptions), with warning indicators for risky states.
  - `<Enter>` adds the highlighted workspace to the selection list and removes it from candidates.
  - Finish keys: `<Ctrl+D>` or typing `done` then `<Enter>`; minimum 1 selection required.
  - Empty input + `<Enter>` does nothing (stays in loop).
  - Filterable list by substring (case-insensitive); lightweight fuzzy match is acceptable.
- Before removal, gathers warnings (e.g., dirty changes, unpushed commits, upstream missing, status unknown) and displays them.
- Before removal, require explicit confirmation for risky states (Dirty/Unpushed/Diverged/Unknown). For multiple selections, the confirmation prompt must mention warnings when present, with stronger wording for Dirty/Unknown.
- When confirmation is shown, include `git status --short --branch` output for repos in warning states under each affected workspace.
- Before removal, if multiple workspaces are selected, ask for confirmation (default No).
- Calls `workspace.RemoveWithOptions`, which:
  - Validates the workspace exists.
  - With confirmation, allows dirty repos to be removed; otherwise fails if any repo has uncommitted/untracked/unstaged/unmerged changes.
  - Runs `git worktree remove --force <worktree>` for dirty workspaces; otherwise runs `git worktree remove <worktree>` for each repo’s worktree.
  - Deletes the workspace directory.

## Success Criteria
- Workspace directory no longer exists; associated worktrees are removed from their bare stores.

## Failure Modes
- Workspace not found.
- Git errors while removing worktrees.
- Filesystem errors while deleting the workspace directory.

---
title: "gws status"
status: implemented
---

## Synopsis
`gws status [<WORKSPACE_ID>]`

## Intent
Summarize the git status of every repo within a workspace to help decide whether it is safe to remove or ship.

## Behavior
- With `WORKSPACE_ID` provided: inspects that workspace.
- Without it: lists available workspaces and interactively prompts for one; fails if none exist.
- Validates workspace existence; then scans its repos and runs `git status --porcelain=v2 -b` in each worktree.
- For each repo, captures branch name, upstream, HEAD short SHA, and counts of staged, unstaged, untracked, unmerged files, plus ahead/behind counts.
- Marks the repo as dirty when any staged/unstaged/untracked/unmerged item exists.
- Renders warnings for status errors per repo and aggregates scan warnings.

## Success Criteria
- Status information printed for each repo in the workspace.

## Failure Modes
- Workspace not found.
- Git errors when running status (reported per repo; fatal errors stop the command).

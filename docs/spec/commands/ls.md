---
title: "gwst ls"
status: implemented
---

## Synopsis
`gwst ls [--details]`

## Intent
List workspaces under `<root>/workspaces` and show a quick view of the repos attached to each.

## Behavior
- Scans `<root>/workspaces` for directories; ignores non-directories.
- For each workspace, scans its contents to discover repo worktrees (alias, repo key, branch, path) and renders them in a tree view.
- If a workspace description is available in `gwst.yaml`, show it alongside the workspace ID.
- If a workspace has status warnings (dirty, unpushed, diverged, unknown), show an inline tag next to the workspace ID (same labels as `gwst rm`).
- Collects and reports non-fatal warnings from scanning workspaces or repos.
- `--details`: include repo-level git status details (same output as `gwst rm` confirmation: `git status --short --branch` for repos that need warnings).

## Success Criteria
- Existing workspaces are listed; command succeeds even if none exist (empty result).

## Failure Modes
- Root path inaccessible or `workspaces/` is not a directory.
- Filesystem or git errors while scanning workspaces (reported as warnings; unrecoverable errors fail the command).

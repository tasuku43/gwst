---
title: "gion repo rm"
status: implemented
---

## Synopsis
`gion repo rm [<repo> ...]`

## Intent
Remove one or more bare repo stores under the gion root when they are unused by any workspace.

## Behavior
- Accepts zero or more repo specs (SSH/HTTPS), same format as `gion repo get`.
  - Example: `git@github.com:owner/repo.git`
- When multiple specs are provided, duplicates are removed while preserving order.
- With no specs and prompts allowed:
  - Loads existing stores via `gion repo ls` and opens a filterable multi-select list.
  - Selection UX matches `gion manifest preset rm`: `<Enter>` selects and removes from candidates; finish with `<Ctrl+D>` or `done`; minimum 1 selection required.
- With no specs and `--no-prompt`, returns an error.
- Before removing:
  - Resolves each repo spec to a canonical repo key (`host/owner/repo`) and store path (`<root>/bare/<host>/<owner>/<repo>.git`).
  - If any explicitly requested repo is missing, fail and make no changes.
  - Scans workspaces under `<root>/workspaces` and finds any workspace referencing a target repo (by normalized repo key).
  - If any references exist, return an error and do not remove anything (no confirmation prompt).
- Removal:
  - Deletes the bare repo directory for each target store.
  - Does not remove parent directories even if empty.
  - Does not modify workspaces or worktrees.

## Success Criteria
- Specified bare repo stores are removed from `<root>/bare`.
- No changes are made when validation fails.

## Failure Modes
- Invalid repo spec.
- No repos exist when running with no args (interactive mode).
- Target repo store not found.
- Repo is referenced by one or more workspaces.
- Filesystem errors while deleting store directories.

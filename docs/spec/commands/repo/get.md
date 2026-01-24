---
title: "gwiac repo get"
status: implemented
migrated_from: "docs/spec/commands/repo-get.md"
---

## Synopsis
`gwiac repo get <repo>`

## Intent
Create or normalize a bare repo store for a remote Git repository.

## Behavior
- Accepts SSH or HTTPS Git URLs (e.g., `git@github.com:owner/repo.git` or `https://github.com/owner/repo.git`).
- Normalizes the repo spec to derive a stable repo key and store path (`<root>/bare/<host>/<owner>/<repo>.git`).
- If the store is missing, clones it as `--bare`.
- Normalizes the store:
  - Sets `remote.origin.fetch` to `+refs/heads/*:refs/remotes/origin/*`.
  - Detects the default branch from the remote and updates `refs/remotes/origin/HEAD` accordingly.
  - Runs `git fetch --prune` when the local store is stale.
  - Prunes local head refs that no longer exist remotely.
## Success Criteria
- Bare store exists, normalized, and up to date with the remote default branch.

## Failure Modes
- Missing repo argument or invalid repo spec.
- Network or git errors during clone/fetch.
- Filesystem errors creating store paths.

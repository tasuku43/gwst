# giongo picker design

## Goal
- Provide a fast way to move into workspaces/worktrees created via fzf or similar tools.
- Use the built-in gion picker to allow flat selection of both workspace rows and repo rows.
- Keep gion focused on IaC workflows; ship the picker as a separate `giongo` binary.

## User experience (CLI + shell)
- `giongo` is interactive and requires a TTY.
- `giongo --print` outputs only the selected **absolute path**.
- The shell wraps `giongo` and performs `cd`.
  - Example (zsh):
    - `giongo() { local dest; dest="$(command giongo --print "$@")" || return $?; [[ -n "$dest" ]] && cd "$dest"; }`

## Candidate sources and scanning
- The source of truth is the **filesystem only** (`workspaces/`).
- List directories under `workspaces/<id>` as workspace candidates.
- Read `workspaces/<id>/.gion/metadata.json` and use `description` for display and search (omit when empty).
- Repo candidates are directories directly under `workspaces/<id>` that have `.git` (exclude `.gion`).
- Use `.git` presence only (do not call `git worktree list`).

## Display and selection
- Keep the existing tree UI:
  - workspace row: `<id> - <description>`
  - repo row: `<repo> (branch: <branch>)`
  - detail lines: `repo: <url>`, `branch: <branch>`, etc.
- Cursor moves across **workspace rows and repo rows** (detail lines are not selectable).
- Selection result:
  - workspace row → `.../workspaces/<id>`
  - repo row → `.../workspaces/<id>/<repo>`
- `giongo` is single-select only.
- Non-TTY runs error out (`TTY required`).

## Search behavior
- One search input filters by:
  - workspace ID
  - workspace description (metadata)
  - repo label (alias/repoKey/branch)
  - repo detail lines (URL/branch, etc.)
- If a repo matches, its parent workspace row is **always shown**.
- If a workspace matches, all repos under it are shown.

## UI implementation direction
- Extend the existing workspace picker.
- Separate tree rendering from selectable rows (workspace/repo).
- Follow `docs/spec/ui/UI.md` for layout and color rules.

## Error handling
- Metadata read failures: treat description as empty (no UI warning).
- Repo `.git` inspection failures: skip that repo.

## Testing
- Unit tests for filtering (description + repo details matching).
- Test that parent workspace remains visible when a repo matches.
- Test path resolution for workspace vs repo selection.
- Test non-TTY error behavior.

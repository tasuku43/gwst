# AGENTS.md — gws repository instructions

## Project summary
- Project: gws (Git Workspace Manager)
- Language: Go
- Goal: Manage workspaces (task-based directories) backed by bare repo stores + git worktrees.

## Non-negotiables (safety)
- Do NOT run destructive commands (e.g., `rm -rf`, `sudo`, `chmod -R`, `dd`, disk operations).
- Do NOT modify files outside the repository root.
- Prefer minimal changes per task. Keep diffs focused.

## Development workflow
- Always run formatting and tests before finishing a task:
    - `gofmt -w .` (or `go fmt ./...` if you prefer)
    - `go test ./...`
- If you change CLI behavior, update docs in `docs/` and task notes if needed.

## Code conventions
- Keep dependencies minimal; prefer Go standard library.
- Use `os/exec` to call `git` (do not use a full Git library in MVP).
- Add clear error messages; propagate underlying `git` stderr when helpful.
- Implement idempotent behavior where practical.

## Repository contracts
- Root resolution precedence:
    1) CLI flag `--root`
    2) env `GWS_ROOT`
    3) `~/.config/gws/config.yaml`
    4) default `~/gws`
- Directory layout under root:
    - `<root>/repos` (bare repo store)
    - `<root>/ws` (workspaces)
- Workspace ID must be a valid Git branch name and equals branch name for worktrees.

## MVP scope
Only implement:
- repo: get / ls
- ws: new / add / ls / status / rm
- gc: dry-run + run (safe guard)
- doctor: minimal checks (stale locks, missing worktrees, inconsistency hints)

If something is outside MVP, stop and document it in `tasks/BACKLOG.md` instead of implementing.

## How to proceed on a task
- Pick one task ID from `tasks/MVP.md`.
- Implement the smallest correct change to satisfy acceptance criteria.
- Add/adjust tests as required.
- Ensure docs remain consistent.

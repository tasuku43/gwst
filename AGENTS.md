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
- For UI implementations, always refer to `docs/UI.md` as the authoritative spec.
- When you know the related issue for a PR, include the issue link/number in the PR body.
- Command specs live in `docs/specs/` (one file per subcommand, YAML frontmatter with `status`):
    - `status: planned` means spec-first discussion; implement only after consensus and flip to `implemented`.
    - New feature/CLI change flow: (1) draft/adjust spec in `docs/specs/<cmd>.md`, (2) review/agree, (3) implement, (4) update spec status, (5) run gofmt/go test.
    - `docs/specs/README.md` indexes specs and describes metadata rules.
    - Quick triage for agents: read only the frontmatter to know if work remains. If `pending` (array) is non-empty, there are unimplemented items even when `status: implemented`. Example to view metadata only: `rg --no-heading -n '^-{3}$' -C2 docs/specs/<cmd>.md` or `sed -n '1,20p' docs/specs/<cmd>.md`.

## Code conventions
- Keep dependencies minimal; prefer Go standard library.
- Use `os/exec` to call `git` (do not use a full Git library in MVP).
- Add clear error messages; propagate underlying `git` stderr when helpful.
- Implement idempotent behavior where practical.

## Repository contracts
- Root resolution precedence:
    1) CLI flag `--root`
    2) env `GWS_ROOT`
    3) default `~/gws`
- Directory layout under root:
    - `<root>/bare` (bare repo store)
    - `<root>/src` (human working tree)
    - `<root>/workspaces` (workspaces)
- Workspace ID must be a valid Git branch name and equals branch name for worktrees.

## MVP scope
Only implement:
- repo: get / ls
- workspace: new / add / ls / status / rm
- doctor: minimal checks (missing remote, non-git workspace entries)

## How to proceed on a task
- Implement the smallest correct change to satisfy acceptance criteria.
- Add/adjust tests as required.
- Ensure docs remain consistent.

## GitHub CLI usage notes
- When creating issues or PRs with `gh`, pass the body via a here-doc to preserve newlines for proper GitHub rendering:
  ```sh
  gh issue create \
    --title "Implement gws create unified command" \
    --body "$(cat <<'EOF'
  ## Summary
  ...
  EOF
  )"
  ```

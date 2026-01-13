# USE CASES (MVP Coverage)

Purpose:
- Enumerate typical ways people use gws and judge whether the current UX is sufficient.
- Make it obvious which subcommands/flags to use for each situation.

Rating legend:
- Excellent: Smooth / matches expectations
- Good: Usable but with clear room to improve
- Fair: Noticeable gaps or manual work
- Missing: Not supported

## Setup / Preparation
- **Initialize root** — `gws init` creates bare/src/workspaces and `templates.yaml` in one shot. Once per environment. Rating: Excellent
- **Define / check templates** — Edit `templates.yaml` directly, confirm names with `gws template ls`. Rating: Good (manual YAML editing, no validation)
- **Fetch repositories** — `gws repo get <repo>` creates bare + src clone; `gws repo ls` lists fetched repos. Rating: Good (does not fetch when bare already exists, so not ideal for updating)
- **Switch roots** — Use `--root` or `GWS_ROOT` to separate environments. Rating: Excellent

## Start / During a Task
- **Create workspace from template** — `gws create --template <name> [<id>]`; prompts if omitted. `workspace_id` becomes the branch name for all repos. Rating: Excellent (interactive repo-get prompt appears when a template repo is missing)
- **Add a repo mid-task** — `gws add <id> <repo>`; branch name = workspace_id, base = origin/HEAD. Rating: Excellent
- **List workspaces** — `gws ls` enumerates workspaces. Rating: Good (minimal detail only)
- **Jump to a path** — `gws path --workspace` or `gws path --src` prints a selected path for `cd`. Rating: Good
- **Check status** — `gws status <id>` shows dirty/untracked counts and HEAD per repo. Rating: Excellent (lightweight)

## Reviews
- **Start a PR review** — `gws create --review <PR URL>` creates `<OWNER>-<REPO>-REVIEW-PR-<num>`, fetches the PR head branch (forks not supported) for GitHub; requires `gh`. Rating: Good (GitHub only)
- **Add repos during review** — Use `gws add` after the review workspace is created. Rating: Good

## Cleanup / Maintenance
- **Safe deletion** — `gws rm <id>` removes worktrees then deletes the workspace dir; refuses when dirty. Rating: Excellent
- **Repo store health check** — `gws doctor [--fix]` detects common remote issues. Rating: Fair (narrow check set)
- **Refresh to latest base** — Forcing a fresh fetch is manual via `git fetch`; gws alone doesn’t cover it. Rating: Fair

## Human + Agent Co-use
- **Separate browsing clone vs task workspaces** — Default layout (bare + `src/` + `workspaces/`) supports this split. Rating: Excellent
- **Non-interactive runs (agent/CI)** — `--no-prompt` suppresses interaction, though destructive ops may still halt. Rating: Good
- **Parallel tasks** — Shared branch name = workspace_id keeps multi-agent parallel work safer. Rating: Excellent

## Known Gaps / Improvement Ideas
- Keeping the bare store “always fresh” depends on manual `git fetch`; no built-in auto/update flow.
- `gws ls` lacks detail; in busy teams it’s hard to get a quick overview.
- `gws doctor` checks only a narrow set; does not catch orphan worktrees, branch conflicts, or stale workspace artifacts.
- No JSON/machine-readable output; agents must parse human text.
- After `gws create --review`, pulling newer PR updates still requires manual fetch; no auto-sync or “refresh review” command.

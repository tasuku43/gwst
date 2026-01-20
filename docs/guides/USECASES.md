# USE CASES (MVP Coverage)

Purpose:
- Enumerate typical ways people use gwst and judge whether the current UX is sufficient.
- Make it obvious which subcommands/flags to use for each situation.

Rating legend:
- Excellent: Smooth / matches expectations
- Good: Usable but with clear room to improve
- Fair: Noticeable gaps or manual work
- Missing: Not supported

## Setup / Preparation
- **Initialize root** — `gwst init` creates bare/workspaces and `gwst.yaml` in one shot. Once per environment. Rating: Excellent
- **Define / check presets** — Edit `gwst.yaml` directly, confirm names with `gwst preset ls`, validate with `gwst preset validate`. Rating: Good
- **Fetch repositories** — `gwst repo get <repo>` creates bare store; `gwst repo ls` lists fetched repos. Rating: Good (does not fetch when bare already exists, so not ideal for updating)
- **Switch roots** — Use `--root` or `GWST_ROOT` to separate environments. Rating: Excellent

## Start / During a Task
- **Create workspace from preset** — `gwst create --preset <name> [<id>]`; prompts if omitted. `workspace_id` becomes the branch name for all repos. Rating: Excellent (interactive repo-get prompt appears when a preset repo is missing)
- **Add a repo mid-task** — `gwst add <id> <repo>`; branch name = workspace_id, base = origin/HEAD. Rating: Excellent
- **List workspaces** — `gwst ls` enumerates workspaces; `gwst ls --details` shows git status lines for warning repos. Rating: Good
- **Jump to a path** — `gwst path --workspace` prints a selected path for `cd`. Rating: Good
- **Check status** — `gwst status <id>` shows dirty/untracked counts and HEAD per repo. Rating: Excellent (lightweight)

## Reviews
- **Start a PR review** — `gwst create --review <PR URL>` creates `<OWNER>-<REPO>-REVIEW-PR-<num>`, fetches the PR head branch (forks not supported) for GitHub; requires `gh`. Rating: Good (GitHub only)
- **Add repos during review** — Use `gwst add` after the review workspace is created. Rating: Good

## Cleanup / Maintenance
- **Safe deletion** — `gwst rm <id>` removes worktrees then deletes the workspace dir; refuses when dirty. Rating: Excellent
- **Repo store health check** — `gwst doctor [--fix]` detects common remote issues. Rating: Fair (narrow check set)
- **Refresh to latest base** — Forcing a fresh fetch is manual via `git fetch`; gwst alone doesn’t cover it. Rating: Fair

## Human + Agent Co-use
- **Non-interactive runs (agent/CI)** — `--no-prompt` suppresses interaction, though destructive ops may still halt. Rating: Good
- **Parallel tasks** — Shared branch name = workspace_id keeps multi-agent parallel work safer. Rating: Excellent

## Known Gaps / Improvement Ideas
- Keeping the bare store “always fresh” depends on manual `git fetch`; no built-in auto/update flow.
- Default `gwst ls` is minimal; use `gwst ls --details` for git status context.
- `gwst doctor` checks only a narrow set; does not catch orphan worktrees, branch conflicts, or stale workspace artifacts.
- No JSON/machine-readable output; agents must parse human text.
- After `gwst create --review`, pulling newer PR updates still requires manual fetch; no auto-sync or “refresh review” command.

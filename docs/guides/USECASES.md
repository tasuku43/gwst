# USE CASES (MVP Coverage)

Purpose:
- Enumerate typical ways people use gion and judge whether the current UX is sufficient.
- Make it obvious which subcommands/flags to use for each situation.

Rating legend:
- Excellent: Smooth / matches expectations
- Good: Usable but with clear room to improve
- Fair: Noticeable gaps or manual work
- Missing: Not supported

## Setup / Preparation
- **Initialize root** — `gion init` creates bare/workspaces and `gion.yaml` in one shot. Once per environment. Rating: Excellent
- **Define / check presets** — Edit `gion.yaml` directly, confirm names with `gion manifest preset ls`, validate with `gion manifest preset validate`. Rating: Good
- **Fetch repositories** — `gion repo get <repo>` creates bare store; `gion repo ls` lists fetched repos. Rating: Good (does not fetch when bare already exists, so not ideal for updating)
- **Switch roots** — Use `--root` or `GION_ROOT` to separate environments. Rating: Excellent

## Start / During a Task
- **Create workspace from preset** — `gion manifest add --preset <name> <id>`; prompts if omitted. By default it updates `gion.yaml` then runs `gion apply`. `workspace_id` becomes the branch name for all repos. Rating: Excellent
- **Add a repo mid-task** — Edit `gion.yaml` to add a `repos[]` entry under the workspace, then run `gion plan` / `gion apply`. Rating: Good
- **List workspaces** — `gion manifest ls` lists inventory and shows drift indicators. Rating: Good
- **Check drift / changes** — `gion plan` shows the full diff between `gion.yaml` and the filesystem. Rating: Good

## Reviews
- **Start a PR review** — `gion manifest add --review <PR URL>` creates `<OWNER>-<REPO>-REVIEW-PR-<num>` inventory and reconciles via apply for GitHub; requires `gh`. Rating: Good (GitHub only)
- **Add repos during review** — Edit `gion.yaml` then reconcile with `gion apply` (same as mid-task). Rating: Good

## Cleanup / Maintenance
- **Safe deletion** — `gion manifest rm <id>` updates inventory then reconciles via apply (which prompts/blocks on destructive risk). Rating: Excellent
- **Repo store health check** — `gion doctor [--fix]` detects common remote issues. Rating: Fair (narrow check set)
- **Refresh to latest base** — Forcing a fresh fetch is manual via `git fetch`; gion alone doesn’t cover it. Rating: Fair

## Human + Agent Co-use
- **Non-interactive runs (agent/CI)** — `--no-prompt` suppresses interaction, though destructive ops may still halt. Rating: Good
- **Parallel tasks** — Shared branch name = workspace_id keeps multi-agent parallel work safer. Rating: Excellent

## Known Gaps / Improvement Ideas
- Keeping the bare store “always fresh” depends on manual `git fetch`; no built-in auto/update flow.
- `gion manifest ls` currently requires a drift model that remains stable and easy to understand.
- `gion doctor` checks only a narrow set; does not catch orphan worktrees, branch conflicts, or stale workspace artifacts.
- No JSON/machine-readable output; agents must parse human text.
- After `gion manifest add --review`, pulling newer PR updates still requires manual fetch; no auto-sync or “refresh review” command.

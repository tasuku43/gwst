title: "gws issue"
status: implemented
pending:
  - interactive-issue-picker
  - multi-issue-workspaces
---

## Synopsis
`gws issue [<ISSUE_URL>] [--workspace-id <id>] [--branch <name>] [--base <ref>] [--no-prompt]`

## Intent
Create a workspace tailored to a single issue, wiring a branch named after the issue and checking it out as a worktree so work can start immediately without manual setup.

## Supported hosts (MVP)
- GitHub, GitLab, Bitbucket issue URLs (e.g., `https://github.com/owner/repo/issues/123`). Other hosts reject with an error.

## Behavior
- If `ISSUE_URL` is provided:
  - Parse the URL to obtain `owner`, `repo`, and `issue number`.
  - Workspace ID: defaults to `ISSUE-<number>`; can be overridden with `--workspace-id`. Must pass `git check-ref-format --branch`. If the workspace already exists, error.
  - Branch: defaults to `issue/<number>`. Before proceeding, prompt the user with the default and allow editing unless `--no-prompt` or `--branch` is supplied.
    - If the branch exists in the bare store, use it.
    - If not, create it from a base ref.
  - Base ref: defaults to the standard detection used by `gws add` (prefer `HEAD`, then `origin/HEAD`, then `main`/`master`/`develop` locally or on origin). `--base` overrides detection; must resolve in the bare store or as `origin/<ref>`.
  - Repo resolution:
    - Only the repo that owns the issue is used (single-repo flow). No template support in MVP.
    - If the repo store is missing, prompt to run `gws repo get <repo>`; with `--no-prompt`, fail.
  - Worktree location: `<root>/workspaces/<WORKSPACE_ID>/<repo_name>`; branch = `issue/<number>`.
  - Existing worktree collision: if the target workspace already has a worktree on the target branch, error (no reuse in MVP).
- If `ISSUE_URL` is omitted and prompts are allowed (interactive picker):
  - `--no-prompt` with no URL => error.
  - Step 1: pick a repo from fetched bare stores whose origin remote resolves to a supported host. Display `alias (host/owner/repo)`; filterable by substring.
  - Step 2: fetch open issues for the chosen repo from the host API (GitHub via `gh api`; other hosts may be added later). Default fetch: latest 50 open issues sorted by updated desc.
  - Step 3: multi-select issues using the same add/remove loop as `gws template new` (filterable list; `<Enter>` adds; `<Ctrl+D>` or `done` to finish; minimum 1 selection).
  - For each selected issue:
    - Workspace ID = `ISSUE-<number>` (no per-item override in this flow).
    - Branch = `issue/<number>`.
    - Base ref detection and repo missing handling are the same as the URL path.
  - Flags `--workspace-id`, `--branch`, and `--base` are only valid when a single issue is targeted (URL path). In picker mode with multiple selections, using these flags is an error.
  - Creation is sequential; an error on one issue stops further creation and reports successes/failures so far.
- Output uses Inputs/Steps/Result (no header line). When multiple workspaces are created, Result lists each workspace/worktree added.

## Success Criteria
- For URL mode: workspace `<root>/workspaces/ISSUE-<number>` exists with a worktree for the issue repo checked out to branch `issue/<number>`.
- For picker mode: each selected issue produces a workspace with the same guarantees; partial success is reported if a later item fails.

## Failure Modes
- Unsupported or invalid issue URL.
- Workspace already exists.
- Repo store missing and user declines/forbids `repo get`.
- Git errors when creating/fetching branch or adding the worktree.
- Base ref cannot be detected.
- Provided `--base` or `--branch` is invalid or conflicts with existing worktree in the target workspace.
- Picker mode: no TTY available, repo selection empty, issue fetch fails (auth/network/API), or zero issues selected.

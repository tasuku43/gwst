---
title: "gwst manifest add"
status: implemented
aliases:
  - "gwst man add"
  - "gwst m add"
pending:
  - interactive-flows
  - error-messages-and-output
---

## Synopsis
`gwst manifest add [--preset <name> | --review [<PR URL>] | --issue [<ISSUE_URL>] | --repo [<repo>]] [<WORKSPACE_ID>] [--branch <name>] [--base <ref>] [--no-apply] [--no-prompt]`

Note: If no mode flag is provided and prompts are allowed, the mode is chosen via an interactive picker.

## Intent
Create the desired workspace inventory in `gwst.yaml` using an interactive UX, then reconcile the filesystem via `gwst apply` by default.

## Modes and selection
- Exactly one of `--preset`, `--review`, `--issue`, or `--repo` can be specified. If multiple are provided, error.
- If none are provided and prompts are allowed, enter an interactive mode picker.
  - The picker presents `preset`, `repo`, `review`, `issue` and supports arrow selection with filterable search.
- If none are provided and `--no-prompt` is set, error.
- When prompts are used, mode flags still run the unified prompt flow so the `Inputs` section is rendered as a single in-place interaction.
- The optional positional `[<WORKSPACE_ID>]` overrides the default workspace ID derivation for single-workspace flows.
  - This is supported only for `--preset` and `--repo` (single-workspace flows).
  - For `--review` and `--issue`, the workspace ID is derived mechanically from the URL metadata; providing `[<WORKSPACE_ID>]` is an error.
  - In multi-select picker flows, providing `[<WORKSPACE_ID>]` is also an error.

### Workspace ID formats (review / issue)
For URL-based modes, workspace IDs are derived mechanically:
- `--review`: `<OWNER>-<REPO>-REVIEW-PR-<number>` (owner/repo uppercased)
- `--issue`: `<OWNER>-<REPO>-ISSUE-<number>` (owner/repo uppercased)

### GitHub-only modes (review / issue)
- `--review` and `--issue` are GitHub-only modes.
- URL parsing and picker flows accept GitHub URLs only.
- These modes require an authenticated GitHub CLI (`gh`) to fetch PR/issue metadata.

## Behavior (high level)
- Runs an interactive selection and input UX (mode picker + mode-specific prompts).
- Produces a workspace definition (mode, description, optional metadata, repo list with alias/repo_key/branch) and writes it to `<root>/gwst.yaml`.
- If the target `WORKSPACE_ID` already exists in `gwst.yaml`, error (no upsert in MVP).
- If the target workspace already exists on the filesystem (`<root>/workspaces/<WORKSPACE_ID>`) but is missing from `gwst.yaml`, error and suggest `gwst import` (do not adopt implicitly).
- By default, runs `gwst apply` to reconcile the filesystem with the updated manifest.
  - `gwst apply` reconciles the entire root (full diff between `gwst.yaml` and filesystem), which may include unrelated drift in the same root.
  - Confirmation and destructive rules follow `gwst apply` spec.
- `--no-prompt` is forwarded to `gwst apply` when apply is run.
  - If the full-root plan contains any removals, `gwst apply --no-prompt` must error; `gwst manifest add` does not preflight-block this case.
- With `--no-apply`, stops after rewriting `gwst.yaml` and prints a suggestion to run `gwst apply` (or `gwst plan`) next.
- When `--no-prompt` is set, all required inputs must be provided via flags/args; missing values are errors (no interactive fallback).
- `--workspace-id` is not supported; use the positional `[<WORKSPACE_ID>]` instead (error if provided).

### Rollback on cancelled apply (prompt decline / Ctrl-C)
When `gwst manifest add` runs apply (default behavior), it performs a two-phase action:
1) rewrite `gwst.yaml`, then
2) run `gwst apply` (which includes the confirmation prompt).

If the user cancels at the apply confirmation step (e.g. answers `n`/No) or interrupts at the prompt (`Ctrl-C`), `gwst manifest add` should roll back the manifest change by restoring the previous `gwst.yaml` content.

Guidance:
- Prefer restoring from a backup of the pre-change `gwst.yaml` (or an in-memory snapshot) rather than running `gwst import`.
  - A simple approach is to write a temporary backup file before the rewrite, then restore it on cancellation.
- Do not use `gwst import` as a rollback mechanism:
  - It can drop desired-state-only entries that are not present on the filesystem.
  - It may lose or change metadata fields not fully reconstructible from the filesystem.
  - It can rewrite the file in ways unrelated to the user's cancelled change.

Notes:
- If apply proceeds past confirmation and starts executing steps, failures are treated as apply failures (the manifest remains updated; users can re-run `gwst apply`).

## Detailed flow (conceptual)
1. Determine mode (flag or interactive picker).
2. Collect inputs (repo/preset selection, URLs, workspace id, description, branches, base).
3. Validate inputs:
   - Mode must be uniquely determined.
   - `WORKSPACE_ID` must be a valid workspace ID (safe directory name). It does not have to be a valid git branch name.
   - `--base` must be `origin/<branch>` when provided.
   - Branch names must be valid git branch names.
4. Collision checks:
   - If `WORKSPACE_ID` exists in `gwst.yaml`, error.
   - If `<root>/workspaces/<WORKSPACE_ID>` exists but is missing from `gwst.yaml`, error and suggest `gwst import`.
5. Rewrite `gwst.yaml` (full-file rewrite).
6. If `--no-apply` is set: stop after manifest rewrite.
7. Otherwise run `gwst apply` for the entire root:
   - This may include unrelated drift in the same root.
   - Confirmation and destructive rules are handled by `gwst apply`.
8. If apply is cancelled at the confirmation step, restore the previous `gwst.yaml`.

## Base ref (`--base`) and default branch behavior
- By default, new branches are created from the repo's default branch (detected from `refs/remotes/origin/HEAD` when available).
- If `--base <ref>` is provided, it must be in the form `origin/<branch>`, and `gwst manifest add` writes it as `base_ref` into the corresponding repo entry in `gwst.yaml`.
  - `base_ref` is used only when the branch does not already exist in the bare store.
  - If `base_ref` does not resolve when it is needed, `gwst apply` fails (manifest remains updated).
- Scope (preset / multi-repo):
   - In `--preset` interactive flows, the command asks for `base_ref` per repo (similar to per-repo branch prompts).
     - The input is pre-filled with the detected default base ref for that repo (typically `origin/<default>` derived from `refs/remotes/origin/HEAD`) so users can press Enter to accept.
     - If detection fails, the default may be empty (meaning "use the repo's default branch").
     - If `--base <ref>` is provided, it is used as the pre-filled default for each repo's base prompt (users can press Enter or edit per repo).
   - With `--no-prompt`, per-repo base selection is not available; `--base <ref>` (when provided) is applied to all repos.

## Branch behavior (`--branch`) and defaults
This command stores the target branch per repo as `repos[].branch` in `gwst.yaml`. When `gwst apply` materializes the workspace, each repo worktree is checked out to that branch.

Defaults and `--branch` rules:
- `--preset`:
  - Default branch for each repo is `<WORKSPACE_ID>`.
  - When prompts are allowed, the command always asks for branch per repo.
    - The input is pre-filled with `<WORKSPACE_ID>` and the cursor is positioned so users can press Enter to accept, or type a suffix (e.g. `-hotfix`) without retyping.
  - With `--no-prompt`, uses the default for all repos (no per-repo override).
- `--repo`:
  - Default branch is `<WORKSPACE_ID>`.
  - When prompts are allowed, the command always asks for the repo branch.
    - The input is pre-filled with `<WORKSPACE_ID>` (or `--branch` when provided) and the cursor is positioned so users can press Enter to accept, or type a suffix without retyping.
  - `--branch <name>` is allowed but does not skip the branch prompt; it is used as the pre-filled default.
  - With `--no-prompt`, `--branch` is optional; when omitted, the default is used.
- `--review`:
  - Branch defaults to the PR head ref (tracking `origin/<head_ref>`).
  - `--branch` is not supported (error if provided).
- `--issue`:
  - Branch defaults to `issue/<number>`.
  - When prompts are allowed, the user is always prompted with the default and can edit it.
    - The input is pre-filled with the default branch (or `--branch` when provided) and the cursor is positioned so users can press Enter to accept, or type a suffix without retyping.
  - `--branch <name>` is allowed but does not skip the branch prompt; it is used as the pre-filled default.
  - `--no-prompt` accepts the default when `--branch` is omitted.

## Multi-selection (review / issue picker)
- When `--review` or `--issue` is used without a URL (picker mode), the UX may allow selecting multiple items.
- In multi-select mode:
  - The command writes all selected workspaces into `gwst.yaml` in one run.
  - Then, by default, runs `gwst apply` once (single plan + single confirmation) for the updated manifest.
  - If any selected item collides (workspace already exists in manifest, or exists on filesystem but is missing from manifest), that item is skipped and reported; other selections proceed (partial success).
  - If all selected items are skipped, the command makes no changes and exits with an error.
  - If at least one item succeeds and at least one item is skipped, the command exits 0 and reports skipped items as warnings (do not fail the whole run).

## Output (IA)
- Always uses the common sectioned layout from `docs/spec/ui/UI.md`.
- `Inputs`: interactive UX inputs (mode, repo/preset, workspace id, branch/base, etc).
- `Plan`/`Apply`/`Result`: delegated to `gwst apply` when apply is run.

### Info section (when apply runs)
When apply runs, `gwst manifest add` should emit an `Info` section after `Inputs` to make the two-phase behavior explicit:
- `manifest: updated gwst.yaml`
- `apply: reconciling entire root (plan may include unrelated drift)`

### Output: `--no-apply`
When `--no-apply` is set, `gwst manifest add` does not run apply and prints a short summary instead.

Example:
```
Inputs
  • mode: repo
  • repo: git@github.com:org/repo.git
  • workspace id: PROJ-123
  • branch: PROJ-123

Result
  • updated gwst.yaml

Suggestion
  gwst apply
```

### Output: with apply (default)
When apply runs, `gwst manifest add` prints `Inputs` first, then streams `gwst apply` output (`Info`/`Plan`/`Apply`/`Result`).
`gwst manifest add` itself does not attempt to summarize the plan beyond what `gwst apply` prints.

## Error messages (guidance)
`gwst manifest add` should keep errors actionable and include the next command when possible.

Common cases:
- Conflicting mode flags: error and mention the allowed set (`--preset`/`--repo`/`--review`/`--issue`).
- Missing mode with `--no-prompt`: error and suggest providing a mode flag.
- Missing required inputs with `--no-prompt`:
  - `--repo` with no repo argument → error.
  - `--issue`/`--review` with no URL → error.
- Workspace already exists in manifest: error and include the workspace id.
- Workspace exists on filesystem but missing in manifest: error and suggest `gwst import`.
- Apply fails after manifest rewrite:
  - Treat as apply failure and keep the manifest change (users can re-run `gwst apply`).

## Success Criteria
- `gwst.yaml` contains the intended workspace entry in normalized form.
- When apply is run and confirmed, filesystem matches the manifest.

## Failure Modes
- Invalid or missing required inputs (subject to prompt rules).
- Manifest write failure.
- `gwst apply` failure (git/filesystem).
  - When apply fails after manifest write, the manifest remains updated; users can re-run `gwst apply`.

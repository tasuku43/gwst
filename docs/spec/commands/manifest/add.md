---
title: "gwst manifest add"
status: planned
aliases:
  - "gwst man add"
  - "gwst m add"
pending:
  - interactive-flows-from-create
  - error-messages-and-output
---

## Synopsis
`gwst manifest add [--preset <name> | --review [<PR URL>] | --issue [<ISSUE_URL>] | --repo [<repo>]] [<WORKSPACE_ID>] [--branch <name>] [--base <ref>] [--no-apply] [--no-prompt]`

## Intent
Create the desired workspace inventory in `gwst.yaml` using an interactive UX (same intent as the legacy `gwst create`), then reconcile the filesystem via `gwst apply` by default.

## Modes and selection
- Exactly one of `--preset`, `--review`, `--issue`, or `--repo` can be specified. If multiple are provided, error.
- If none are provided and prompts are allowed, enter an interactive mode picker.
  - The picker presents `preset`, `repo`, `review`, `issue` and supports arrow selection with filterable search.
- If none are provided and `--no-prompt` is set, error.
- When prompts are used, mode flags still run the unified prompt flow so the `Inputs` section is rendered as a single in-place interaction.
- The optional positional `[<WORKSPACE_ID>]` overrides the default workspace ID derivation for single-workspace flows.
  - This applies to URL-based modes as well (e.g. `--review <PR URL> [<WORKSPACE_ID>]`, `--issue <ISSUE_URL> [<WORKSPACE_ID>]`).
  - In multi-select picker flows, providing `[<WORKSPACE_ID>]` is an error.

## Behavior (high level)
- Runs the same interactive selection and input UX as the legacy `gwst create` (mode picker + mode-specific prompts).
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

## Detailed flow (conceptual)
1. Determine mode (flag or interactive picker).
2. Collect inputs (repo/preset selection, URLs, workspace id, description, branches, base).
3. Validate inputs:
   - Mode must be uniquely determined.
   - `WORKSPACE_ID` must be a valid git branch name.
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

## Base ref (`--base`) and default branch behavior
- By default, new branches are created from the repo's default branch (detected from `refs/remotes/origin/HEAD` when available).
- If `--base <ref>` is provided, it must be in the form `origin/<branch>`, and `gwst manifest add` writes it as `base_ref` into the corresponding repo entry in `gwst.yaml`.
  - `base_ref` is used only when the branch does not already exist in the bare store.
  - If `base_ref` does not resolve when it is needed, `gwst apply` fails (manifest remains updated).
 - Scope (preset / multi-repo):
   - In `--preset` mode, `--base` applies to the entire workspace and is written into every repo entry as `repos[].base_ref`.
   - Per-repo base selection is not supported in MVP; users can edit `gwst.yaml` manually if needed.

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
  - When prompts are allowed, the command asks for the repo branch.
    - The input is pre-filled with `<WORKSPACE_ID>` and the cursor is positioned so users can press Enter to accept, or type a suffix without retyping.
  - `--branch <name>` overrides the default and skips the branch prompt.
  - With `--no-prompt`, `--branch` is optional; when omitted, the default is used.
- `--review`:
  - Branch defaults to the PR head ref (tracking `origin/<head_ref>`).
  - `--branch` is not supported (error if provided).
- `--issue`:
  - Branch defaults to `issue/<number>`.
  - When prompts are allowed and `--branch` is not provided, the user is prompted with the default and can edit it.
    - The input is pre-filled with the default branch and the cursor is positioned so users can press Enter to accept, or type a suffix without retyping.
  - `--branch <name>` overrides the default and skips the branch prompt.
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

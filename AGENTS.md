# AGENTS.md — gwst repository instructions

## Project summary
- Project: gwst (Git Workspace Manager)
- Language: Go
- Goal: Manage workspaces (task-based directories) backed by bare repo stores + git worktrees.

## IaC-first direction (Plan / Apply / Import)
- gwst uses an IaC-style workflow centered on **Declare → Diff → Reconcile**.
- `gwst.yaml` is the workspace **desired state (inventory)**:
  - Location: `<GWST_ROOT>/gwst.yaml`
  - Inventory is updated via `gwst manifest ...`.
  - `gwst apply`: `gwst.yaml` is the source of truth; gwst computes a diff and reconciles the filesystem.
  - `gwst import`: the filesystem (+ `.gwst/metadata.json`) is the source of truth; gwst rebuilds `gwst.yaml`.
- Command roles:
  - `gwst plan`: **read-only** (no side effects). Shows the diff between `gwst.yaml` and the filesystem.
  - `gwst apply`: **reconcile**. Output is `Plan` → (confirm y/n) → `Apply` → `Result`
    - The confirmation prompt is shown at the end of the `Plan` section (with a blank line before the prompt).
    - `Apply` prints steps plus partial git command logs (tree-style).
    - `Result` prints a completion summary (applied counts / manifest rewrite, etc.).
  - `gwst import`: **rebuild inventory**. Reconstructs/normalizes `gwst.yaml` from the current filesystem.
- Non-negotiables:
  - Idempotent (running `apply` repeatedly converges to no changes)
  - Destructive changes require explicit confirmation (`--no-prompt` must not allow destructive changes)
  - Treat drift detection (`plan`) and restoration (`apply`/`import`) as first-class
- Specs (source of truth):
  - `docs/spec/core/GWST.md`
  - `docs/spec/commands/plan.md`, `docs/spec/commands/apply.md`, `docs/spec/commands/import.md`
  - `docs/spec/ui/UI.md`

## Plans / Backlogs
- Design notes and backlogs live under `docs/plans/`.
- Main IaC backlog: `docs/plans/2026-01-21-iac-backlog.md`

## Non-negotiables (safety)
- Do NOT run destructive commands (e.g., `rm -rf`, `sudo`, `chmod -R`, `dd`, disk operations).
- Do NOT modify files outside the repository root.
- Prefer minimal changes per task. Keep diffs focused.

## Development workflow
- Always run formatting, tests, vet, and build before finishing a task:
    - `gofmt -w .` (or `go fmt ./...` if you prefer)
    - `go test ./...`
    - `go vet ./...`
    - `go build ./...`
- If you change CLI behavior, update docs in `docs/` and task notes if needed.
- For UI implementations, always refer to `docs/spec/ui/UI.md` as the authoritative spec.

## CLI output conventions (important)
- Keep the section order from `docs/spec/ui/UI.md` and avoid emitting duplicate sections.
- Follow the color semantics from `docs/spec/ui/UI.md` (success/warn/error/muted/accent) when adding colored output; do not introduce command-specific color rules without updating the spec.
- Tree/list indentation must use shared tokens from `internal/infra/output` (e.g. `output.Indent`, `output.Indent2`, `output.TreeBranch*`, `output.TreeStem*`) to keep nesting consistent across commands.
- For manifest mutation commands (`gwst manifest add/rm/gc/...`):
  - Do not print "partial" output (e.g. an `Info` section) before calling `applyManifestMutation`.
  - Instead, compute everything up front and render via `applyManifestMutation` hooks:
    - `ShowPrelude` for user-provided inputs (interactive selections / flag-driven args).
    - `RenderInfoBeforeApply` for derived metadata (warnings, scanned/candidates counts, computed candidate lists, etc.).
  - Rationale: emitting sections before `applyManifestMutation` often leads to ordering drift (`Info` before `Inputs`) or duplicated `Info` sections.
- When you know the related issue for a PR, include the issue link/number in the PR body(e.g. Fixes #<issue-number>).
- Command specs live in `docs/spec/commands/` (YAML frontmatter with `status`):
    - `status: planned` means spec-first discussion; implement only after consensus and flip to `implemented`.
    - Layout mirrors CLI: `docs/spec/commands/<cmd>.md` or `docs/spec/commands/<cmd>/<sub>.md` (+ optional `docs/spec/commands/<cmd>/README.md`)
    - New feature/CLI change flow: (1) draft/adjust spec, (2) review/agree, (3) implement, (4) update spec status, (5) run gofmt/go test.
    - `docs/spec/README.md` indexes specs and describes metadata rules.
    - Quick triage for agents: read only the frontmatter to know if work remains. If `pending` (array) is non-empty, there are unimplemented items even when `status: implemented`. Example: `sed -n '1,20p' docs/spec/commands/<cmd>.md` or `sed -n '1,20p' docs/spec/commands/<cmd>/<sub>.md`.

## Code conventions
- Keep dependencies minimal; prefer Go standard library.
- Use `os/exec` to call `git` (do not use a full Git library in MVP).
- Add clear error messages; propagate underlying `git` stderr when helpful.
- Implement idempotent behavior where practical.

## Repository contracts
- Root resolution precedence:
    1) CLI flag `--root`
    2) env `GWST_ROOT`
    3) default `~/gwst`
- Directory layout under root:
    - `<root>/bare` (bare repo store)
    - `<root>/src` (human working tree)
    - `<root>/workspaces` (workspaces)
- Workspace ID must be a valid Git branch name and equals branch name for worktrees.

## MVP scope
Only implement:
- repo: get / ls
- manifest: ls / add / rm / preset *
- reconcile: plan / apply / import
- open
- doctor: minimal checks (missing remote, non-git workspace entries)

## How to proceed on a task
- Implement the smallest correct change to satisfy acceptance criteria.
- Add/adjust tests as required.
- Ensure docs remain consistent.

## GitHub CLI usage notes
- When creating issues or PRs with `gh`, pass the body via a here-doc to preserve newlines for proper GitHub rendering:
  ```sh
  gh issue create \
    --title "Update gwst command surface" \
    --body "$(cat <<'EOF'
  ## Summary
  ...
  EOF
  )"
  ```

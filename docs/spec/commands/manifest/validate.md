---
title: "gion manifest validate"
status: implemented
aliases:
  - "gion man validate"
  - "gion m validate"
---

## Synopsis
`gion manifest validate [--root <path>] [--no-prompt]`

## Intent
Validate `gion.yaml` (the desired-state inventory) for schema correctness and core invariants, producing actionable errors suitable for humans and CI.

This is a read-only command. It does not scan the filesystem and does not run `gion apply`.

## Behavior
- Loads `<root>/gion.yaml`; missing or unreadable file is reported as an issue.
- Parses YAML and reports errors if invalid.
- Validates top-level structure:
  - `version` is optional; when present must be `1`.
  - `workspaces` mapping must exist.
- Validates each workspace entry under `workspaces`:
  - Workspace IDs must satisfy git branch ref format rules (`git check-ref-format --branch`).
  - Workspace entries must be a mapping.
  - `mode` is optional; when present must be one of: `preset`, `repo`, `review`, `issue`, `resume`, `add`.
  - If `mode=preset`, `preset_name` must be non-empty.
  - `source_url` is optional; when present it must be a valid absolute URL (scheme + host).
  - `repos` must exist and be a list (it may be empty).
- Validates each repo entry in `workspaces.<id>.repos[]`:
  - Entry must be a mapping.
  - `alias`, `repo_key`, `branch` are required and must be non-empty strings.
  - `alias` must be unique within the workspace and must not be `.gion`.
  - `repo_key` must be in the form `<host>/<owner>/<repo>` or `<host>/<owner>/<repo>.git`.
  - `branch` must satisfy git branch ref format rules (`git check-ref-format --branch`).
  - `base_ref` is optional; when present must be `origin/<branch>` and `<branch>` must satisfy git branch ref format rules.
- Presets:
  - Preset entries are validated by `gion manifest preset validate`.
  - This command may include preset-related issues in the same output.
- Output uses the standard sectioned layout:
  - `Result` contains one bullet per issue; when no issues are found, prints `no issues found`.
- Exit status:
  - exit 0 when no issues are found.
  - exit 1 when one or more issues are found.
- `--no-prompt` is accepted but has no effect (kept for CLI consistency).

## Output example
```
Result
  • workspaces: missing required field
  • workspaces.PROJ-123.repos[0].branch: invalid branch name (contains space)
  • workspaces.PROJ-123.repos[1].alias: duplicate alias "api"
  • workspaces.PROJ-123.repos[2].repo_key: invalid repo key (must be host/owner/repo[.git])
```

## Related behavior
- `gion plan` must fail (non-zero exit) when `gion.yaml` is invalid; it must not print a plan in that case.

## Success Criteria
- Returns success when `gion.yaml` is valid.

## Failure Modes
- `gion.yaml` missing/unreadable.
- YAML parse error.
- Invalid or missing required fields.

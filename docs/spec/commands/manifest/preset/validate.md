---
title: "gwst manifest preset validate"
status: planned
aliases:
  - "gwst manifest pre validate"
  - "gwst manifest p validate"
pending:
  - validation-rules-and-output
---

## Synopsis
`gwst manifest preset validate [--no-prompt]`

## Intent
Validate preset entries in `gwst.yaml`.

## Notes
- This is the manifest-first replacement for the legacy `gwst preset validate`.
- This command is inventory-only and does not run `gwst apply`.

## Behavior
- Loads `<root>/gwst.yaml`; missing or unreadable file is reported as an issue.
- Parses YAML and reports errors if invalid.
- Checks for required fields:
  - top-level `presets` mapping exists.
  - each preset entry includes a non-empty `repos` list.
- Detects duplicate preset names in the YAML source.
- Validates preset names using the same rules as `gwst manifest preset add`.
- Validates each repo spec via the existing repo spec normalization rules.
- Output uses the standard sectioned layout:
  - `Result` contains one bullet per issue; when no issues are found, prints `no issues found`.
- Exit status:
  - exit 0 when no issues are found.
  - exit 1 when one or more issues are found.
- `--no-prompt` is accepted but has no effect (kept for CLI consistency).

## Output example
```
Result
  • presets.helpdesk.repos: missing or empty
  • presets.\"bad name\".name: invalid (allowed: [A-Za-z0-9_-], len 1-64)
  • presets.helpers.repos[1]: invalid repo spec (parse error)
```

## Success Criteria
- Returns success when `gwst.yaml` presets are valid.

## Failure Modes
- `gwst.yaml` missing/unreadable.
- YAML parse error.
- Missing required fields, duplicate preset names, invalid preset names, or invalid repo specs.

---
title: "gwst template validate"
status: implemented
---

## Synopsis
`gwst template validate`

## Intent
Validate `templates.yaml` to catch malformed templates before use.

## Behavior
- Loads `<root>/templates.yaml`; missing or unreadable file is reported as an issue.
- Parses YAML and reports errors if invalid.
- Checks for required fields:
  - top-level `templates` mapping exists.
  - each template entry includes a non-empty `repos` list.
- Detects duplicate template names in the YAML source.
- Validates template names using the same rules as `gwst template add`.
- Validates each repo spec via the existing repo spec normalization rules.
- Output uses the standard “Result” section with one bullet per issue; when no issues are found, prints “no issues found”.

## Success Criteria
- Returns success when `templates.yaml` is valid.

## Failure Modes
- `templates.yaml` missing/unreadable.
- YAML parse error.
- Missing required fields, duplicate template names, invalid template names, or invalid repo specs.

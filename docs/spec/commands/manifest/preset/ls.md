---
title: "gwst manifest preset ls"
status: implemented
aliases:
  - "gwst manifest pre ls"
  - "gwst manifest p ls"
---

## Synopsis
`gwst manifest preset ls [--no-prompt]`

## Intent
List preset entries in `gwst.yaml`.

## Notes
- This command is inventory-only and does not run `gwst apply`.

## Behavior
- Loads `<root>/gwst.yaml`; fails if the file is missing, unreadable, or invalid YAML.
- Parses preset entries and prints them in sorted order by preset name.
- For each preset, lists its repository specs in the stored order.
- No changes are made (read-only).
- `--no-prompt` is accepted but has no effect (kept for CLI consistency).

## Output
Uses the common sectioned layout. No interactive UI is required.

- `Info` (optional): preset count.
- `Result`: preset list.

Example:
```
Info
  • presets: 2

Result
  • helpdesk
    ├─ git@github.com:org/repo.git
    └─ git@github.com:org/api.git
  • helpers
    └─ git@github.com:org/tooling.git
```

## Success Criteria
- Presets are listed with their repo entries; if none exist, the command reports that no presets were found.

## Failure Modes
- Root not resolved.
- `gwst.yaml` missing, unreadable, or invalid YAML.

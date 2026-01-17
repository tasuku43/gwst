---
title: "gws CLI specs"
status: implemented
---

This directory holds command-level specifications in English. Each file uses frontmatter metadata to track implementation status so we can evolve features intentionally.

## Metadata rules
- Required: `title`, `status` (`implemented` or `planned`).
- Optional: `pending` (YAML array of short tokens/ids for unimplemented pieces). If `pending` is non-empty, treat the spec as needing work even when `status: implemented`.
- Additional optional fields (e.g., `description`, `since`) are allowed.
- Use YAML frontmatter at the top of each spec.

## Global CLI behavior
- Command form: `gws <command> [flags] [args]`.
- Root resolution precedence: `--root` flag > `GWS_ROOT` environment variable > default `~/gws`.
- Common flags: `--root <path>`, `--no-prompt`, `--debug`, `--help`/`-h`.
- Output: human-readable text only in the current MVP; JSON output is future work.

## Debug logging
- `--debug` enables debug logging to a file (no on-screen debug output).
- Output directory: `<GWS_ROOT>/logs/`.
- File naming: `debug-YYYYMMDD.log` using local date.
- Append mode: always append to the day file (single file per day).
- Format: one event per line, human-readable key/value pairs.
- Required fields: `ts`, `pid`, `trace`, `kind`, `phase`.
- Optional fields: `ws` (workspace id when known), `cmd`, `line`, `code`, `prompt`, `step`, `step_id`.
- `trace` is generated per command execution.
- `kind` values: `cmd`, `stdout`, `stderr`, `exit`.
- `phase` values: `inputs`, `info`, `steps`, `result`, `prompt`, `none`.
- `prompt` is used only when `phase=prompt` (e.g. `prompt=workspace-id`).
- `step`/`step_id` are used only when `phase=steps` (both are set).

Example:
```
ts=2026-01-17T12:34:56.789-08:00 pid=12345 trace=git:abcd phase=steps step=2 step_id=repo-get kind=cmd cmd="git fetch origin main"
ts=2026-01-17T12:34:56.912-08:00 pid=12345 trace=git:abcd phase=steps step=2 step_id=repo-get kind=stdout line="..."
ts=2026-01-17T12:34:57.003-08:00 pid=12345 trace=git:abcd phase=steps step=2 step_id=repo-get kind=stderr line="..."
ts=2026-01-17T12:34:57.010-08:00 pid=12345 trace=git:abcd phase=steps step=2 step_id=repo-get kind=exit code=0
ts=2026-01-17T12:35:01.000-08:00 pid=12345 trace=exec:beef phase=prompt prompt=workspace-id kind=cmd cmd="gh api -X GET ..."
```

## Spec files
Current command specs live in this folder:
- `docs/specs/init.md`
- `docs/specs/doctor.md`
- `docs/specs/repo-get.md`
- `docs/specs/repo-ls.md`
- `docs/specs/template-ls.md`
- `docs/specs/template-add.md`
- `docs/specs/template-rm.md`
- `docs/specs/resume.md`
- `docs/specs/create.md`
- `docs/specs/add.md`
- `docs/specs/ls.md`
- `docs/specs/status.md`
- `docs/specs/rm.md`
- `docs/specs/open.md`
- `docs/specs/path.md`

Add new files in the same format when introducing new commands or options.

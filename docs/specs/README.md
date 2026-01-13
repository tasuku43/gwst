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
- Common flags: `--root <path>`, `--no-prompt`, `--verbose`/`-v`, `--help`/`-h`.
- Output: human-readable text only in the current MVP; JSON output is future work.

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

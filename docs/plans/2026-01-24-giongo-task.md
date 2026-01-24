# giongo implementation tasks

## Status
- overall: not started

## Scope
- Add a separate `giongo` binary focused on interactive workspace/worktree navigation.
- Reuse the existing picker UI and scanning logic from gion where possible.
- Keep gion's IaC command surface unchanged.

## Tasks
- [ ] Spec and docs
- [ ] CLI entrypoint (`cmd/giongo`)
- [ ] Picker model extension (workspace + repo selectable rows)
- [ ] Filesystem scanning + metadata description support
- [ ] Search filtering (workspace + repo + details)
- [ ] Selection output (absolute path, `--print`)
- [ ] Non-TTY error behavior
- [ ] GoReleaser config (builds + archives for `giongo`)
- [ ] Tests (filtering, parent visibility, path resolution, non-TTY)
- [ ] README/INSTALL updates

## Notes
- Update task statuses as work progresses.

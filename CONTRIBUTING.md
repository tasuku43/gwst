# Contributing to gion

Thanks for your interest in contributing.

## Quick links

- Docs index: `docs/README.md`
- Specs (contracts): `docs/spec/README.md`
- Support / questions: `SUPPORT.md`

## Development setup

Prerequisites:
- Go **1.24+**
- Git

Clone:

```bash
git clone https://github.com/tasuku43/gion.git
cd gion
```

Run tests:

```bash
go test ./...
```

Common checks (recommended before opening a PR):

```bash
gofmt -w .
go vet ./...
go test ./...
go build ./...
```

Optional (Taskfile):

```bash
task fmt
task test
task build:dev
```

## Running locally

```bash
go run ./cmd/gion --help
go run ./cmd/giongo --help
```

## Submitting changes

- Keep PRs small and focused (one change per PR is ideal).
- Add/adjust tests when it makes sense.
- If you change CLI behavior or output, update the relevant docs under `docs/` (especially command specs and UI conventions).
- Follow existing code style and patterns (standard `gofmt` formatting).

## Issues / feature requests

- Bugs and feature requests: GitHub Issues (see `SUPPORT.md`).
- Questions and general help: GitHub Discussions (see `SUPPORT.md`).

## Release notes (maintainers)

Releases are driven by Git tags and GitHub Actions. See:
- `docs/ops/RELEASING.md`
- `docs/ops/RELEASE_PIPELINE.md`

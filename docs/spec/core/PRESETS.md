---
title: "gwiac.yaml presets"
status: implemented
---

# gwiac.yaml presets

`gwiac.yaml` includes a `presets` section that defines named groups of repositories for creating workspaces.

## Location

`<GWIAC_ROOT>/gwiac.yaml`

Create the file (and the default directory layout) with:

```
gwiac init
```

## Format

Top-level key is `presets`. Each preset has a `repos` list.

```yaml
presets:
  example:
    repos:
      - git@github.com:org/backend.git
      - git@github.com:org/frontend.git
      - git@github.com:org/manifests.git
      - git@github.com:org/docs.git
```

Notes:
- Repo specs must be SSH (`git@host:owner/repo.git`) or HTTPS (`https://host/owner/repo.git`).
- `gwiac manifest preset validate` checks YAML structure, preset names, and repo spec format.

## CLI usage

Create a workspace from a preset:

```
gwiac manifest add --preset example MY-123
```

List presets:

```
gwiac manifest preset ls
```

Add/remove presets without editing YAML directly:

```
gwiac manifest preset add mytmpl --repo git@github.com:org/repo.git
gwiac manifest preset rm mytmpl
```

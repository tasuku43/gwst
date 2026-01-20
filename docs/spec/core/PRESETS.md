---
title: "gwst.yaml presets"
status: implemented
---

# gwst.yaml presets

`gwst.yaml` includes a `presets` section that defines named groups of repositories for creating workspaces.

## Location

`<GWST_ROOT>/gwst.yaml`

Create the file (and the default directory layout) with:

```
gwst init
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
- `gwst preset validate` checks YAML structure, preset names, and repo spec format.

## CLI usage

Create a workspace from a preset:

```
gwst create --preset example MY-123
```

List presets:

```
gwst preset ls
```

Add/remove presets without editing YAML directly:

```
gwst preset add mytmpl --repo git@github.com:org/repo.git
gwst preset rm mytmpl
```

---
title: "templates.yaml"
status: implemented
---

# templates.yaml

`templates.yaml` defines named groups of repositories that can be used to create a workspace in one command.

## Location

`<GWST_ROOT>/templates.yaml`

Create the file (and the default directory layout) with:

```
gwst init
```

## Format

Top-level key is `templates`. Each template has a `repos` list.

```yaml
templates:
  example:
    repos:
      - git@github.com:org/backend.git
      - git@github.com:org/frontend.git
      - git@github.com:org/manifests.git
      - git@github.com:org/docs.git
```

Notes:
- Repo specs must be SSH (`git@host:owner/repo.git`) or HTTPS (`https://host/owner/repo.git`).
- `gwst template validate` checks YAML structure, template names, and repo spec format.

## CLI usage

Create a workspace from a template:

```
gwst create --template example MY-123
```

List templates:

```
gwst template ls
```

Add/remove templates without editing YAML directly:

```
gwst template add mytmpl --repo git@github.com:org/repo.git
gwst template rm mytmpl
```

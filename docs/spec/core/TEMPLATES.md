---
title: "templates.yaml"
status: implemented
---

# templates.yaml

`templates.yaml` defines named groups of repositories that can be used to create a workspace in one command.

## Location

`<GWS_ROOT>/templates.yaml`

Create the file (and the default directory layout) with:

```
gws init
```

## Format

Top-level key is `templates`. Each template has a `repos` list.

```yaml
templates:
  example:
    repos:
      - git@github.com:octocat/Hello-World.git
      - https://github.com/octocat/Spoon-Knife.git
```

Notes:
- Repo specs must be SSH (`git@host:owner/repo.git`) or HTTPS (`https://host/owner/repo.git`).
- `gws template validate` checks YAML structure, template names, and repo spec format.

## CLI usage

Create a workspace from a template:

```
gws create --template example MY-123
```

List templates:

```
gws template ls
```

Add/remove templates without editing YAML directly:

```
gws template add mytmpl --repo git@github.com:org/repo.git
gws template rm mytmpl
```

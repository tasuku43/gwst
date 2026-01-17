# UI Specification (Bubble Tea + Bubbles + Lip Gloss)

## Goals
- Provide a quiet, consistent output experience (similar to Codex CLI)
- Use Bubble Tea for interactive flows; use plain text for non-interactive flows
- Keep the information architecture consistent across all commands

## Scope
- Interactive (TTY): Bubble Tea + Bubbles + Lip Gloss
- Non-interactive (non-TTY): plain text (no TUI), same layout rules

## Layout (common)
Sectioned layout. Inputs appear first when present; Info/Suggestion are optional:

```
Inputs
  <input line>
  <input line>

Info
  <warnings / skipped / blocked>

Steps
  <step line>
  <step line>

Result
  <result line>
  <tree>

Suggestion
  <next command>
```

Rules:
- Indent: 2 spaces
- 1 blank line between sections
- Section order is fixed: Inputs → Info → Steps → Result → Suggestion
- Result lines are bullets (use the same prefix as Steps)
- No success banner; success is implied in Result section

## Prefix & Indentation
- Default prefix token: `•` (can be changed later)
- Steps/list lines are prefixed with `2 spaces + prefix + space`
- Result lines are prefixed with `2 spaces + prefix + space`

Prefix coloring:
- Info/list (Steps, Results): prefix is muted gray
- Prompts/questions (e.g. run now?): prefix + label use accent (cyan)
- Optional details (repo lines under workspace): muted gray

Example:
```
Steps
  • repo get git@github.com:org/repo.git -> bare/github.com/org/repo.git
  • worktree add repo -> workspaces/PROJ-123/repo
```

## Command execution logs
- Steps may include user-facing command logs.
- Use muted color (low contrast, non-flashy).
- Command output lines can be visually connected using tree glyphs.
- Debug logging is written to files when `--debug` is provided (no on-screen Debug section).

Example:
```
Steps
  • repo get git@github.com:org/repo.git
    └─ $ git clone --bare ...
```

## Colors (fixed)
- success: green
- warn: yellow
- error: red
- muted/log: low-contrast gray
- accent/meta: cyan (for metadata like branch)

## Components (Bubble Tea)
- Text input: Bubbles textinput
- Select/confirm: Bubbles list or simple radio
- Spinners: optional; use subtle style only
- Help line: muted color, minimal content
- Long selection lists should scroll so Inputs stay visible.

## Examples

### gws create --template (interactive)
```
Inputs
  • template: hel
    └─ helmfiles
  • workspace id: SREP-123

Info
  • (optional warnings / skipped / blocked)

Steps
  • worktree add helmfiles

Result
  • /Users/me/gws/workspaces/SREP-123
    └─ helmfiles (branch: SREP-123)
```

### gws create --template (non-interactive)
```
Steps
  • repo get git@github.com:org/repo.git
  • worktree add repo

Result
  • /Users/me/gws/workspaces/SREP-123
    ├─ repo
    └─ api
```

### gws create --review (interactive)
```
Steps
  • repo get required for 1 repo
    └─ gws repo get git@github.com:org/repo.git
  • run now? (y/n)
```

### gws status
```
Result
  • api (branch: PROJ-123)
    ├─ head: 94a67ef
    ├─ staged: 1
    ├─ unstaged: 2
    └─ untracked: 2
```

## Notes
- Prefix token is a theme value and can be changed later.
- All prompts and labels are English.
- Info section is optional and may include warnings/skipped/blocked items, selection state, and inline help/auxiliary meta.
- Errors should be emphasized with a red prefix and red text.
- Suggestion section is optional and shown only on TTY with colors enabled (e.g. `cd <path>`).

## Implementation contract
- CLI output must use `ui.Renderer` (or `internal/core/output` helpers) and must not write directly to stdout via `fmt.Fprintf/Printf/Println` in UI paths.
- Result lines must be rendered using `Bullet()` to enforce consistent prefixing.

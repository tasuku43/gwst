# UI Specification (Bubble Tea + Bubbles + Lip Gloss)

Status: draft (MVP+)

## Goals
- Codex CLI のように静かで一貫した出力体験を提供する
- 対話は Bubble Tea に統一し、非対話はプレーン出力で整える
- 出力の情報設計は全コマンドで統一する

## Scope
- Interactive (TTY): Bubble Tea + Bubbles + Lip Gloss
- Non-interactive (non-TTY): plain text (no TUI), same layout rules
- JSON/format switching: MVP では扱わない

## Layout (common)
Sectioned layout with 3 sections (interactive may insert Inputs):

```
Header

Inputs
  <input line>
  <input line>

Steps
  <step line>
  <step line>

Result
  <result line>
  <tree>
```

Rules:
- Indent: 2 spaces
- 1 blank line between sections
- No success banner; success is implied in Result section

## Prefix & Indentation
- Default prefix token: `•` (can be changed later)
- Steps/list lines are prefixed with `2 spaces + prefix + space`
- Result lines are prefixed with `2 spaces`

Prefix coloring:
- Info/list (Steps, Results): prefix is muted gray
- Prompts/questions (e.g. run now?): prefix + label use accent (cyan)
- Optional details (repo lines under workspace): muted gray

Example:
```
Steps
  • repo get git@github.com:org/repo.git
  • worktree add repo
```

## Command execution logs
- Logs are included in Steps (not only verbose).
- Use muted color (low contrast, non-flashy).
- Command output lines can be visually connected using tree glyphs.

Example:
```
Steps
  › repo get git@github.com:org/repo.git
    └─ $ git clone --bare ...
    └─ $ git clone ... /src/...
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

## Examples

### gws new (interactive)
```
gws new

Inputs
  • template: hel
    └─ helmfiles
  • workspace id: SREP-123

Steps
  • worktree add helmfiles

Result
  /Users/me/gws/ws/SREP-123
    └─ helmfiles (branch: SREP-123)
```

### gws new (non-interactive)
```
gws new (template: app)

Steps
  › repo get git@github.com:org/repo.git
  › worktree add repo

Result
  /Users/me/gws/ws/REVIEW-PR-123
    ├─ repo
    └─ api
```

### gws review (interactive)
```
gws review <PR URL>

Steps
  • repo get required for 1 repo
    └─ gws repo get git@github.com:org/repo.git
  • run now? (y/n)
```

## Notes
- Prefix token is a theme value and can be changed later.
- All prompts and labels are English.

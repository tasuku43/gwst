---
title: "UI Architecture"
status: implemented
---

# UI Architecture

This document describes the implementation structure (separation of concerns) that keeps the UI aligned with the UI.md contract.

## Goals
- Always guarantee the fixed section order: Inputs → Info → Steps → Result → Suggestion
- Avoid per-command bespoke rendering; enforce the contract through shared components
- Prevent duplicate Inputs output during interactive flows (update in-place instead)

## Components

### Frame (`internal/ui/frame.go`)
**Responsibilities**
- A single container that centralizes section ordering and rendering rules
- Holds `Inputs/Info/Steps/Result/Suggestion` content and renders them in a fixed order

**Usage**
- `SetInputsPrompt(...)` sets prompt lines
- `AppendInputsRaw(...)` appends already-formatted list/tree lines
- `SetInfo(...)` / `AppendInfoRaw(...)` manage auxiliary info

**Key points**
- Frame owns the screen structure
- Each UI updates only the content

### Renderer (`internal/ui/renderer.go`)
**Responsibilities**
- Low-level rendering of headers, bullets, steps, and tree lines
- Invoked by Frame or CLI render paths

### Prompt Models (`internal/ui/prompt.go`)
**Responsibilities**
- Input/selection state transitions and validation
- `View()` should only update Inputs/Info via Frame

## Implementation Rules
- Do not print UI output directly with `fmt.Fprintf/Printf/Println` (use Renderer/Frame)
- Consolidate prompts into `Inputs`, and auxiliary info into `Info`
- Do not invent custom headers (e.g., “Selected”); fold them into `Info`
- Do not use AltScreen; keep CLI output non-invasive (`tea.WithAltScreen` is not allowed)

## Applying to Existing Flows
- Prefer a single Frame that updates across multiple prompt steps
- Use `AppendInputsRaw(...)` for list/tree lines to keep the section order intact

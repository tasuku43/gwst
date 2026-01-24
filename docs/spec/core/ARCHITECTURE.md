---
title: "Architecture"
status: planned
---

# Architecture

This document describes cross-cutting design principles that apply across commands and subsystems.

## Application layer reuse

Shared application logic should be implemented in a reusable layer so it can be invoked by:
- explicit commands (e.g., `gwiac import`)
- implicit rebuilds after `create`/`rm`/`add`/`resume`

CLI commands should focus on argument parsing and presentation, delegating the actual operations to the shared layer.

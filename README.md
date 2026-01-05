# gws — Git Workspaces for Human + Agentic Development

gws is a tool that shifts local development in the agentic era from “clone-directory centric” to “workspace centric.”

## What it solves

Traditional local development assumes one repository equals one clone directory edited by humans. In a world where multiple AI agents work in parallel on the same machine, this creates bottlenecks:

- Weak isolation by task (artifact mixing, mistakes)
- Poor control over worktree creation/removal (higher risk)
- No clear overview of which workspace belongs to which task, making cleanup (GC) scary

gws keeps a bare repo as the “master” and ensures all work happens in worktrees under workspaces.

## Quickstart (assumed)

```bash
# Root (optional)
export GWS_ROOT=~/work   # Defaults to ~/gws if unset

# Create a workspace (workspace_id must be a valid branch name)
gws new PROJ-1234

# Add repos (create worktrees under the workspace)
gws add PROJ-1234 git@github.com:org/backend.git --alias backend
gws add PROJ-1234 git@github.com:org/frontend.git --alias frontend

# Check status
gws status PROJ-1234

```

See `docs/` for details.

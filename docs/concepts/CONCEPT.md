# gws Concept

## 1. Why gws is needed (Problem)

In the era of AI agents, multiple actors (humans + multiple agents) make changes in parallel on the same machine and the same codebase. With the traditional workflow of directly editing a single clone directory, the following issues become prominent:

- Context collisions (changes and generated artifacts from different tasks get mixed)
- The number of working directories grows beyond what humans can reliably organize (it becomes unclear what each directory is for)
- Cleanup (deletion) feels risky, so leftovers accumulate and the environment becomes even more error-prone
- Agents are more likely to perform destructive operations by mistake (e.g., `rm -rf`, running commands in the wrong directory)

gws promotes working directories into explicit **workspaces (task-scoped working directories)** and enables Git worktrees to be operated in a **standardized, safer, and listable** way.  
gws focuses on **creating and managing work environments**; downstream development workflows (run, test, PR, etc.) are intentionally left to the user’s existing practices.

## 2. Who this tool is for (Target users)

gws is primarily for developers and teams who work in the terminal and build productivity by composing existing tools (git/gh/tmux/make/just/direnv, etc.).

- Prefer tracking state via text and commands rather than delegating the workflow to a GUI
- Run multiple tasks (humans + agents) in parallel, but struggle with isolation and cleanup
- Need to treat changes across multi-repo / monorepo setups as a single “task unit”
- Want to apply the same workspace concept in remote development and CI environments (future expansion)

## 3. What it provides (Minimal primitives)

gws provides a minimal set of management capabilities: **create, extend, list, and safely clean up** workspaces.  
It does not replace existing development practices; instead, it makes workspaces **composable** with the user’s preferred toolchain and command workflows.

## 4. When it helps (Representative scenarios)

- Run humans and multiple agents in parallel without contaminating each other’s working contexts
- Group changes spanning multiple repositories into a single task unit (workspace)
- Quickly spin up multiple environments for PR review or reproduction, and safely dispose of them

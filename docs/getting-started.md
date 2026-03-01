# Getting Started

## Prerequisites

Noodle orchestrates AI coding agents inside tmux sessions. You need:

- **tmux** — terminal multiplexer. Install with `brew install tmux` (macOS) or your system package manager.
- **Claude Code** or **Codex CLI** — at least one AI coding agent. Noodle spawns these as cooks that do the actual work.
- **Git** — Noodle operates on git repositories and uses worktrees for isolation.

## Install Noodle

```sh
brew install poteto/tap/noodle
```

Verify the install:

```sh
noodle --version
```

## Initialize a Project

Open a terminal in an existing git repository and run:

```sh
noodle start
```

On first run, Noodle creates the project structure if it does not already exist:

```
brain/
  index.md          # Brain vault index
  todos.md          # Backlog — your task list
  principles.md     # Project principles the agent follows
.noodle/            # Runtime state (gitignored)
.noodle.toml        # Configuration
```

## What Each File Does

**`.noodle.toml`** — project configuration. Sets the default model provider, skills path, and runtime options. The generated default:

```toml
mode = "auto"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
```

**`brain/todos.md`** — your backlog. Numbered markdown checkboxes that Noodle's scheduler reads to decide what to work on.

**`brain/principles.md`** — guidelines the agent follows. Write project-specific rules here (coding style, testing requirements, deploy process).

**`.noodle/`** — runtime state. Contains orders, session data, and mise (the scheduler's snapshot of project state). Gitignored by default.

## Add a Backlog Item

Edit `brain/todos.md`:

```markdown
# Todos

<!-- next-id: 2 -->

## Backlog

1. [ ] Add a /healthz endpoint that returns 200 OK
```

The `<!-- next-id: N -->` comment tracks the next available ID. Increment it each time you add an item.

## Watch Noodle Work

With a backlog item and skills in place, `noodle start` begins the loop:

1. **Schedule** — the scheduler skill reads `brain/todos.md` and writes orders to `.noodle/orders-next.json`. Each order describes what to do and which skill handles it.
2. **Cook** — Noodle spawns an AI agent (a "cook") in a tmux session to execute the order. The cook runs the assigned skill, makes code changes, and commits.
3. **Merge** — completed work is merged back. The loop re-schedules and picks up the next item.

Open the web UI at `localhost:3000` to see active sessions, order progress, and session summaries.

## Review the Output

After a cook finishes:

- **Commits** appear on the branch the cook worked in. Noodle uses worktrees to isolate concurrent work.
- **Session summaries** are available in the web UI and in `.noodle/` runtime state.
- **Backlog updates** — if the cook marked items done, `brain/todos.md` reflects the changes.

Run `noodle status` to see the current loop state: active orders, running cooks, and pending work.

## Next Steps

- [Skills](/concepts/skills) — how skills work and how to write your own
- [Scheduling](/concepts/scheduling) — how the LLM-powered scheduler decides what to do
- [Brain](/concepts/brain) — the persistent memory vault
- [Configuration](/reference/configuration) — all config options
- [Examples](/examples) — working example projects to copy and adapt

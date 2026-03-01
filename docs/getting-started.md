# Getting Started

Noodle is a skill-based agent orchestration framework. You teach agents what to do by writing skills — markdown files with a prompt and some metadata. A scheduling agent decides what to work on, execution agents do the work, and the system learns from each session. Skills are the only extension point. Read the full [Vision](/vision) to understand the design.

## Install

Ask your agent to set up Noodle for you. Point it at the [Install](/install) page — it covers the binary, skills, and backlog configuration.

Or do it yourself:

```sh
brew install poteto/tap/noodle
```

You'll also need **Claude Code** or **Codex CLI** (at least one) and **Git**.

## Init a project

`cd` into an existing git repo and run:

```sh
noodle start
```

On first run, Noodle creates the project structure for you:

```
brain/
  index.md
  todos.md
  principles.md
.noodle/            # runtime state (gitignored)
.noodle.toml        # configuration
```

## What the files do

**`.noodle.toml`** -- project configuration. Controls the default model, skills path, and runtime mode. The generated default looks like this:

```toml
mode = "auto"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
```

`mode = "auto"` means Noodle runs the full schedule-cook-merge loop on its own. The skills path tells Noodle where to find your skill definitions.

**`brain/todos.md`** -- your backlog. Numbered markdown checkboxes that the scheduler reads to decide what to work on next.

**`brain/principles.md`** -- guidelines the agent follows when working. Put project-specific rules here: coding style, testing requirements, deploy process, whatever matters to you.

**`.noodle/`** -- runtime state. Orders, session data, and the scheduler's snapshot of project state live here. Gitignored by default, so you never commit it.

**`.agents/skills/`** -- where your skills live. Each subdirectory contains a `SKILL.md` that defines what the skill does, when it runs, and how. This is the main extension point in Noodle.

## Write your first skill

Skills are how you teach Noodle what to do. Each skill is a directory under `.agents/skills/` with a `SKILL.md` file. The file has YAML frontmatter (name, description, schedule) followed by markdown instructions the agent reads when executing the skill.

Create an execute skill -- the one that actually implements backlog items:

```sh
mkdir -p .agents/skills/execute
```

Write `.agents/skills/execute/SKILL.md`:

```yaml
---
name: execute
description: Implements a backlog item. Reads the task prompt, makes changes, commits.
schedule: "When backlog items with linked plans are ready for implementation"
---

# Execute

Implement the task described in the prompt. Make the code changes, verify they work, and commit with a conventional commit message.

## Steps

1. Read the task description from the prompt.
2. Make the required changes.
3. Verify: run tests or checks relevant to the change.
4. Commit with a message in the format: `<type>(<scope>): <description>`.
```

The `schedule` field is a natural language description. Noodle's LLM-powered scheduler reads it to decide when this skill should run.

You'll probably want a schedule skill too -- the one that reads the backlog and produces work orders:

```yaml
---
name: schedule
description: Reads backlog and produces work orders for the loop.
schedule: "When orders are empty or after backlog changes"
---

# Schedule

Read `brain/todos.md` for backlog items. Write `.noodle/orders-next.json` with orders for each unchecked item.
```

Two skills is enough to get started. The scheduler reads the backlog, writes orders, and the execute skill picks them up.

## Add a backlog item

Edit `brain/todos.md`:

```markdown
# Todos

<!-- next-id: 2 -->

## Backlog

1. [ ] Add a /healthz endpoint that returns 200 OK
```

The `<!-- next-id: N -->` comment tracks the next available ID. Increment it each time you add an item.

## Run `noodle start` and watch it work

With skills defined and a backlog item ready, run:

```sh
noodle start
```

The loop works in three phases:

1. **Schedule** -- the scheduler skill reads `brain/todos.md` and writes orders to `.noodle/orders-next.json`. Each order says what to do and which skill handles it.
2. **Cook** -- Noodle spawns an AI agent (a "cook") as a child process. The cook runs the assigned skill, makes code changes in an isolated worktree, and commits.
3. **Merge** -- completed work merges back. The loop re-schedules and picks up the next item.

This keeps going until the backlog is empty or you stop it.

## Review the output

After a cook finishes:

- **Commits** appear on the branch the cook worked in. Each cook gets its own worktree, so concurrent work stays isolated.
- **Web UI** at `localhost:3000` shows active sessions, order progress, and session summaries.
- **Backlog updates** -- if the cook marked items done, `brain/todos.md` reflects the changes.

Run `noodle status` to see the current loop state from the terminal: active orders, running cooks, and pending work.

## Next steps

- [Vision](/vision) -- the design philosophy behind skill-based orchestration
- [Skills](/concepts/skills) -- how skills work and how to write more of them
- [Scheduling](/concepts/scheduling) -- how the LLM-powered scheduler decides what to do
- [Brain](/concepts/brain) -- the persistent memory vault
- [Configuration](/reference/configuration) -- all config options
- [Cookbook](/cookbook/) -- patterns and recipes to copy and adapt

# Getting Started

Noodle is a skill-based agent orchestration framework. You teach agents what to do by writing skills -- markdown files with a prompt and some metadata. A scheduling agent decides what to work on, execution agents do the work, and the system merges the output. Skills are the only extension point. Read the full [Vision](/vision) to understand the design.

## Install

### Binary

```sh
brew install poteto/tap/noodle
```

Verify the install:

```sh
noodle --version
```

No Homebrew? Download the binary from [GitHub releases](https://github.com/poteto/noodle/releases) and put it on your `PATH`.

### Prerequisites

You need **Git** and at least one agent CLI:

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) -- `claude`
- [Codex CLI](https://github.com/openai/codex) -- `codex`

Noodle spawns these as child processes. Make sure whichever you use is installed and authenticated.

## Key terms

These come up throughout the docs. See the [Glossary](/glossary) for the full list.

- **Skill** -- a markdown file that teaches an agent how to do something. Noodle's only extension point.
- **Task type** -- a skill with a `schedule` field. Runs autonomously in the loop.
- **Chef** -- the scheduling agent. Reads the backlog, writes orders.
- **Cook** -- an execution agent. Picks up an order, works in an isolated worktree.
- **Orders** -- work assignments written by the chef.
- **Mise** -- the project snapshot the chef reads before scheduling (backlog state, active orders, history).
- **Worktree** -- a git worktree. Each cook gets its own so concurrent agents never conflict.

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

**`brain/todos.md`** -- your backlog. Numbered markdown checkboxes that the scheduler reads to decide what to work on next. This is the default backlog source, but Noodle also supports adapters that sync work from GitHub Issues, Linear, Jira, or other external trackers. See [Adapters](/concepts/adapters) for details.

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

## Recommended skills by project type

**Any project** -- start with these:

| Skill | Purpose | Task type? |
|-------|---------|------------|
| `schedule` | Reads backlog, writes work orders | Yes |
| `execute` | Implements tasks, commits changes | Yes |
| `commit` | Conventional commit formatting | No |

**Projects that want quality gates** -- add:

| Skill | Purpose | Task type? |
|-------|---------|------------|
| `quality` | Reviews completed work before merge | Yes |
| `testing` | Runs test suite against changes | Yes |
| `review` | Code review walkthrough | No |

**Projects with complex tasks** -- add:

| Skill | Purpose | Task type? |
|-------|---------|------------|
| `plan` | Breaks down large tasks into phased plans | No |
| `debugging` | Systematic root-cause analysis | No |

**Projects that want self-improvement** -- add [brainmaxxing](https://github.com/poteto/brainmaxxing):

| Skill | Purpose | Task type? |
|-------|---------|------------|
| `reflect` | Writes session learnings to the brain | Yes |
| `meditate` | Distills principles from accumulated learnings | Yes |

Task-type skills (those with a `schedule` field) run autonomously in the loop. General skills get invoked directly by agents during execution.

## Configure routing

Edit `.noodle.toml` to set the default model and any tag overrides:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"
```

If your project uses multiple models for different tasks:

```toml
[routing.tags.fast]
provider = "claude"
model = "claude-sonnet-4-6"

[routing.tags.review]
provider = "claude"
model = "claude-opus-4-6"
```

The scheduling agent can reference these tags when creating orders. See [Configuration](/reference/configuration) for all options.

## Add a backlog item

Edit `brain/todos.md`:

```markdown
# Todos

<!-- next-id: 2 -->

## Backlog

1. [ ] Add a /healthz endpoint that returns 200 OK
```

The `<!-- next-id: N -->` comment tracks the next available ID. Increment it each time you add an item.

## A note on cost

Each cook is a full LLM session. Running `max_cooks = 4` means up to four concurrent API sessions, each consuming tokens independently. Start with `max_cooks = 1` or `2` in your `.noodle.toml` while you're learning the system:

```toml
[concurrency]
max_cooks = 2
```

Scale up once you've seen the cost per session on your workload.

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
- **Web UI** at `localhost:3000` shows a live event feed, the order queue, and stage status for each active session.
- **Backlog updates** -- if the cook marked items done, `brain/todos.md` reflects the changes.

Run `noodle status` to see the current loop state from the terminal: active orders, running cooks, and pending work.

## Next steps

- [Vision](/vision) -- the design philosophy behind skill-based orchestration
- [Skills](/concepts/skills) -- how skills work and how to write more of them
- [Scheduling](/concepts/scheduling) -- how the LLM-powered scheduler decides what to do
- [Brain](/concepts/brain) -- the optional persistent memory vault
- [Glossary](/glossary) -- quick reference for Noodle terminology
- [Configuration](/reference/configuration) -- all config options
- [Cookbook](/cookbook/) -- patterns and recipes to copy and adapt

# Minimal Example

The smallest Noodle setup. Demonstrates the core loop: schedule, execute, commit.

## What's Here

```
.noodle.toml                    # Config: routing defaults, skills path
.agents/skills/schedule/SKILL.md  # Reads backlog, writes orders
.agents/skills/execute/SKILL.md   # Implements a task and commits
brain/todos.md                  # Backlog with two items
```

## How It Works

1. Noodle starts and finds no pending orders.
2. The **schedule** skill reads `brain/todos.md` and creates an order for each unchecked item.
3. The **execute** skill picks up each order, implements it, and commits.
4. The loop re-schedules. When all items are done, it idles.

## Running

```sh
noodle start
```

Noodle reads `.noodle.toml`, discovers the two skills, and begins cycling. Open the web UI (default `localhost:3000`) to watch progress.

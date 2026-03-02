# Multi-Skill Example

Extends the minimal example with custom task types. Demonstrates multi-stage order pipelines with routing defaults.

## What's Here

```
.noodle.toml                      # Config with routing defaults and concurrency
.agents/skills/schedule/SKILL.md  # Builds multi-stage orders (execute -> test -> deploy)
.agents/skills/execute/SKILL.md   # Implements a task and commits
.agents/skills/test/SKILL.md      # Runs the test suite (custom task type)
.agents/skills/deploy/SKILL.md    # Deploys changes (custom task type)
brain/todos.md                    # Backlog with three items
brain/principles.md               # Project principles the agent follows
```

## How It Works

The `test` and `deploy` skills have `schedule` frontmatter, which registers them as task types. The scheduler can then include them as stages in order pipelines.

A typical order pipeline:

1. **execute** — implement the backlog item
2. **test** — run the test suite against the changes
3. **deploy** — ship the verified changes

Each stage runs sequentially within an order. The loop advances to the next stage only after the current one completes.

The config sets project-wide routing defaults and concurrency for multi-stage execution.

## Running

```sh
noodle start
```

Watch the web UI at `localhost:3000` to see multi-stage orders progress through their pipelines.

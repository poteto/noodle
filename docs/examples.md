# Examples

Working example projects live in the [`examples/`](https://github.com/noodle-run/noodle/tree/main/examples) directory. Each is a self-contained Noodle project you can copy and run.

## Minimal

**[`examples/minimal/`](https://github.com/noodle-run/noodle/tree/main/examples/minimal)**

The smallest setup. Two skills (schedule and execute), a two-item backlog, and a config file. Demonstrates the core loop: the scheduler reads the backlog, creates orders, and the execute skill implements each one.

Files:

| File | Purpose |
|------|---------|
| `.noodle.toml` | Routing defaults, skills path |
| `.agents/skills/schedule/SKILL.md` | Reads backlog, writes orders |
| `.agents/skills/execute/SKILL.md` | Implements a task and commits |
| `brain/todos.md` | Backlog with two items |

## Multi-Skill

**[`examples/multi-skill/`](https://github.com/noodle-run/noodle/tree/main/examples/multi-skill)**

Builds on the minimal example with custom task types. Adds `test` and `deploy` skills that have `schedule` frontmatter, registering them as task types the scheduler can include in order pipelines.

A typical order pipeline: execute, then test, then deploy. Each stage runs sequentially within an order.

Files:

| File | Purpose |
|------|---------|
| `.noodle.toml` | Routing defaults, tag overrides, concurrency |
| `.agents/skills/schedule/SKILL.md` | Builds multi-stage orders |
| `.agents/skills/execute/SKILL.md` | Implements a task and commits |
| `.agents/skills/test/SKILL.md` | Runs the test suite (custom task type) |
| `.agents/skills/deploy/SKILL.md` | Deploys verified changes (custom task type) |
| `brain/todos.md` | Backlog with three items |
| `brain/principles.md` | Project principles |

This example also demonstrates routing tag overrides in `.noodle.toml` — routing review-tagged work to a different model.

## Running an Example

```sh
cd examples/minimal   # or examples/multi-skill
noodle start
```

Noodle reads `.noodle.toml`, discovers skills, and begins the schedule-execute loop. Open the web UI at `localhost:3000` to watch progress.

## Prerequisites

- [tmux](https://github.com/tmux/tmux)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) or [Codex CLI](https://github.com/openai/codex)
- Noodle installed — see [Getting Started](/getting-started)

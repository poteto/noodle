# Installing Noodle

This is for agents setting up Noodle in a project. Follow these steps in order.

## Prerequisites

- **tmux** — Noodle spawns agent sessions in tmux. Install it if you don't have it.
- **Claude Code or Codex CLI** — at least one must be available.

## Install the binary

macOS (Homebrew):

```sh
brew install poteto/tap/noodle
```

Verify it works:

```sh
noodle --help
```

## What is Noodle

Noodle is an AI coding framework. Agents do the work, humans steer. The only extension point is skills, markdown files that teach agents how to do things. No plugins, no hooks API, no config DSL.

All state lives in plain files. `.noodle/orders.json` is the work orders. `.noodle/mise.json` is gathered project state. Session logs, control commands, verdicts. All just files on disk. `cat` them, `jq` them, `git diff` them. Nothing is hidden.

It follows a kitchen brigade. You're the Chef. Strategy and review. A scheduling agent reads the project state and decides what to cook next. Cooks do the work in isolated worktrees. A quality gate checks their output. The loop: gather state, schedule, cook, review, merge.

You can run one agent or twenty in parallel. The scheduler picks the right model for each task.

## Set up your project

Install the noodle skill so your agent can reference it for configuration details:

```sh
cp -r /path/to/noodle/.agents/skills/noodle .agents/skills/noodle
```

The noodle skill at `.agents/skills/noodle/SKILL.md` covers:
- Config schema (`.noodle.toml`) — every field, type, default, description
- CLI commands — full reference table
- Adapter setup — how to wire your backlog system (markdown, GitHub Issues, Linear, custom)
- Hook installation — brain injection, auto-index
- Skill management — installing skills, search path precedence
- Troubleshooting — diagnostics, common fixes

Read the skill before configuring anything. It has the details this document intentionally doesn't repeat.

## Start

```sh
noodle start
```

First run detects a fresh project and scaffolds what's missing:
- `brain/` directory with starter templates (index, todos, principles, plans)
- `.noodle/` runtime state directory
- `.noodle.toml` with minimal defaults

After scaffolding, the agent handles the parts that need judgment: choosing adapters, installing skills, wiring hooks. The binary just does the mechanical work.

`noodle start` is always safe to run. Existing files are never overwritten.

## Learn more

[PHILOSOPHY.md](PHILOSOPHY.md) explains why Noodle works this way.

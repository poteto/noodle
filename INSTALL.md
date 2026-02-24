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

Noodle is an AI coding framework where agents do the work and humans steer. Everything runs through skills — markdown files that teach agents how to do things. There are no plugins, no hooks API, no configuration DSL. Skills are the single extension point.

All state lives in files. `.noodle/queue.json` is the work queue. `.noodle/mise.json` is the gathered project state. Verdicts, session logs, control commands — all files. Any tool can read them, any agent can write to them. There's no hidden state and nothing to reverse-engineer.

Noodle follows a kitchen brigade model. The human is the Chef — you set strategy and review output. The scheduling agent (Prioritize) reads the project state and decides what to work on next. Cooks pick up tasks and do the work in isolated worktrees. Quality reviews what they produce. The loop runs continuously: gather state, schedule, cook, review, merge.

Agents work on branches and merge to main when done. You can run one agent or twenty in parallel. The scheduling agent picks the right model for each task — expensive models for hard problems, cheap ones for routine work.

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

After scaffolding, the agent handles everything that requires judgment: choosing adapters, installing skills, setting up hooks. The binary handles the mechanical parts.

`noodle start` is always safe to run. It's idempotent — existing files are never overwritten.

## Learn more

Read [PHILOSOPHY.md](PHILOSOPHY.md) for the deeper rationale behind the brain, self-learning, and why Noodle is built around agent autonomy.

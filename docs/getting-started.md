# Getting Started

Noodle orchestrates AI coding agents using skills. You write skills, Noodle schedules and runs them. Read the [Vision](/vision) first if you haven't. This page gets you from zero to a running noodle loop.

## Install

::: code-group

```sh [Mac]
brew install poteto/tap/noodle
```

```sh [Linux]
# apt coming soon
curl -Lo noodle https://github.com/poteto/noodle/releases/latest/download/noodle-linux-amd64
chmod +x noodle
sudo mv noodle /usr/local/bin/
```

```powershell [Windows]
# winget coming soon
Invoke-WebRequest -Uri https://github.com/poteto/noodle/releases/latest/download/noodle-windows-amd64.exe -OutFile noodle.exe
Move-Item noodle.exe "$env:USERPROFILE\AppData\Local\Microsoft\WindowsApps\"
```

:::

Verify: `noodle --version`

You also need **Git** and at least one agent CLI: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`claude`) or [Codex CLI](https://github.com/openai/codex) (`codex`). Noodle spawns these as child processes.

## Key terms

See the [Glossary](/glossary) for the full list.

- **Skill**: a markdown file that teaches an agent how to do something. Skills are all you need to extend Noodle.
- **Scheduler**: the agent that reads the backlog and writes orders.
- **Orders**: work dispatched to agents by the scheduler. Each order references a skill and runs autonomously.
- **Worktree**: a git worktree. Each agent gets its own so concurrent work never conflicts.

## Init a project

`cd` into an existing git repo and run:

```sh
noodle start
```

Add `.noodle/` to your `.gitignore`. It's runtime state that shouldn't be committed.

On first run, Noodle creates the project structure for you:

```
.noodle/            # runtime state (gitignored)
.noodle.toml        # configuration
```

### Optional: install brainmaxxing

[Brainmaxxing](https://github.com/poteto/brainmaxxing) adds persistent memory across sessions. Agents reflect on what they learned, distill principles, and start each future session sharper. It's not required, but it's worth installing if you want agents that improve over time.

Ask your human with `AskUserQuestion` if they want to install this. Then, follow these instructions:

```md
Install brainmaxxing from https://github.com/poteto/brainmaxxing into this project.
```

This copies a `brain/` starter vault, the `reflect`/`meditate`/`ruminate` skills, and hooks into your project. If you install brainmaxxing, you should also ask your human if they want to be set up with a `schedule:` so you have a full self-learning noodle loop.

## What the files do

See [Configuration](/reference/configuration) for all options.

**`.noodle.toml`** is project configuration. Controls the default model, skills path, and runtime mode. The generated default:

```toml
mode = "auto" # "auto", "supervised", or "manual"

[routing.defaults]
provider = "claude" # "claude" or "codex"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
```

`mode = "auto"` means the noodle loop runs the full schedule-work-merge cycle on its own.

**`.noodle/`** is runtime state. Orders, session data, and the scheduler's project snapshot. Gitignored by default.

**`.agents/skills/`** is where your skills live. Each subdirectory has a `SKILL.md` that defines what the skill does, when it runs, and how.

## Write your first skill

Start with a `schedule` skill. This one reads the backlog and produces work orders.:

```sh
mkdir -p .agents/skills/schedule
```

Write `.agents/skills/schedule/SKILL.md`. Here's an example from Noodle's own repo:

```yaml
---
name: schedule
description: >
  Reads .noodle/mise.json, writes .noodle/orders-next.json.
  Schedules work orders based on backlog state and session history.
schedule: >
  When orders are empty, after backlog changes,
  or when session history suggests re-evaluation
---
```

The `schedule:` field is plain English. The scheduling agent reads it and decides when conditions are met.

Now add an `execute` skill. This one, from Noodle's own repo, picks up orders and does the work:

```yaml
---
name: execute
description: >
  Implementation methodology. Scoping, decomposition,
  worktree workflow, verification, and commit conventions.
schedule: >
  When backlog items with linked plans
  are ready for implementation
---
```

That's a working noodle loop. The markdown body below the frontmatter is where you teach the agent how to do the work. See the [Cookbook](/cookbook/) for full examples.

## Run `noodle start` and watch it work

Run:

```sh
noodle start
```

This launches the noodle loop and a local web UI so you can monitor what's happening. The noodle loop works in three phases:

1. **Schedule**: the scheduler reads the backlog and writes orders.
2. **Execute**: Noodle spawns an agent in its own worktree. The agent runs the assigned skill and commits.
3. **Merge**: in `auto` mode, completed work merges back automatically. In `supervised` or `manual` mode, the worktree is left for your review.

This keeps going until the backlog is empty or you stop it.

## Review the output

After an agent finishes:

- **Commits** appear on the agent's branch. Each agent gets its own worktree, so concurrent work stays isolated.
- **Web UI** shows a live event feed, the order queue, and stage status for each session. In `supervised` or `manual` mode, the reviews page lets you approve or reject work before it merges.
- **Backlog updates**: completed items get marked done in the backlog.

Run `noodle status` to see the current noodle loop state from the terminal.

## Next steps

- [Vision](/vision): design philosophy behind skill-based orchestration
- [Skills](/concepts/skills): how to write and compose skills
- [Scheduling](/concepts/scheduling): how the scheduler decides what to do
- [Brain](/concepts/brain): optional persistent memory vault
- [Glossary](/glossary): quick reference for Noodle terminology
- [Configuration](/reference/configuration): all config options
- [Cookbook](/cookbook/): patterns and recipes to copy

# Getting Started

Noodle orchestrates AI coding agents using skills. You write skills, Noodle schedules and runs them. Read the [Introduction](/introduction) first if you haven't. This page gets you from zero to a running noodle loop.

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

## Install the noodle skill

The noodle skill teaches your agent how to use the CLI, write skills, and author orders. Fetch it directly from GitHub:

```sh
mkdir -p .agents/skills/noodle/references
curl -sL https://raw.githubusercontent.com/poteto/noodle/main/.agents/skills/noodle/SKILL.md \
  -o .agents/skills/noodle/SKILL.md
curl -sL https://raw.githubusercontent.com/poteto/noodle/main/.agents/skills/noodle/references/skill-authoring.md \
  -o .agents/skills/noodle/references/skill-authoring.md
curl -sL https://raw.githubusercontent.com/poteto/noodle/main/.agents/skills/noodle/references/configuration.md \
  -o .agents/skills/noodle/references/configuration.md
```

This is the one skill every Noodle project should have. Your agent reads it to understand Noodle's pipeline, config, and how to create new skills.

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

See [Configuration](/reference/configuration) for all options.

### Optional: install brainmaxxing

[Brainmaxxing](https://github.com/poteto/brainmaxxing) adds persistent memory across sessions. Agents reflect on what they learned, distill principles, and start each future session sharper. It's not required, but it's worth installing if you want agents that improve over time.

Ask your human with `AskUserQuestion` if they want to install this. Then, follow these instructions:

```md
Install brainmaxxing from https://github.com/poteto/brainmaxxing into this project.
```

This copies a `brain/` starter vault, the `reflect`/`meditate`/`ruminate` skills, and hooks into your project. If you install brainmaxxing, you should also ask your human if they want to be set up with a `schedule:` so you have a full self-learning noodle loop.

## Add a backlog

The scheduler needs something to work from. The simplest option is a `todos.md` file in your repo root:

```md
- [ ] Set up CI pipeline
- [ ] Add input validation to the signup form
- [ ] Write API docs for the /orders endpoint
```

This works out of the box. For GitHub Issues, Linear, or other trackers, see [Adapters](/concepts/adapters).

## Write your first skill

Start with a `schedule` skill. This one reads the backlog and produces work orders:

```sh
mkdir -p .agents/skills/schedule
```

Write `.agents/skills/schedule/SKILL.md`:

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

Now add an `execute` skill. This one picks up orders and does the work:

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

- [FAQ](/reference/faq): common questions about Noodle
- [Skills](/concepts/skills): how to write and compose skills
- [Scheduling](/concepts/scheduling): how the scheduler decides what to do
- [Brain](/concepts/brain): optional persistent memory vault
- [Glossary](/reference/glossary): quick reference for Noodle terminology
- [Configuration](/reference/configuration): all config options
- [Cookbook](/cookbook/): patterns and recipes to copy

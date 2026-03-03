# Getting Started

Noodle orchestrates AI coding agents using skills. You write skills, Noodle schedules and runs them. Read the [Introduction](/introduction) first if you haven't. This page gets you from zero to a running noodle loop.

## Install

Give this prompt to your coding agent ([Claude Code](https://docs.anthropic.com/en/docs/claude-code), [Codex CLI](https://github.com/openai/codex), etc.):

```md
Install Noodle and set up this project. Follow the instructions at
https://raw.githubusercontent.com/poteto/noodle/main/INSTALL.md
```

The agent installs the binary, creates a config, writes schedule and execute skills tailored to your project, seeds a backlog, and gets the loop running.

::: details Manual install

::: code-group

```sh [Mac]
brew install poteto/tap/noodle
```

```sh [Linux]
curl -Lo noodle https://github.com/poteto/noodle/releases/latest/download/noodle-linux-amd64
chmod +x noodle
sudo mv noodle /usr/local/bin/
```

```powershell [Windows]
Invoke-WebRequest -Uri https://github.com/poteto/noodle/releases/latest/download/noodle-windows-amd64.exe -OutFile noodle.exe
Move-Item noodle.exe "$env:USERPROFILE\AppData\Local\Microsoft\WindowsApps\"
```

:::

Verify: `noodle --version`

You also need **Git** and at least one agent CLI. Then follow the steps in [INSTALL.md](https://github.com/poteto/noodle/blob/main/INSTALL.md) manually.

:::

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

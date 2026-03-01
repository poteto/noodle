# Install Noodle

This page is for agents setting up Noodle in a project. If you're a human, see [Getting Started](/getting-started).

## Install the binary

```sh
brew install poteto/tap/noodle
```

Verify:

```sh
noodle --version
```

No Homebrew? Download the binary from [GitHub releases](https://github.com/poteto/noodle/releases) and put it on your `PATH`.

## Initialize the project

Run this from the root of a git repository:

```sh
noodle start --once
```

This creates the project structure and runs one scheduling cycle. It will create:

```
.noodle.toml        # configuration
.noodle/            # runtime state (gitignored)
brain/
  index.md          # vault index
  todos.md          # backlog
  principles.md     # project principles
```

The `--once` flag exits after one cycle instead of running the loop continuously. Useful for initial setup.

## Set up skills

Skills are how you teach Noodle what to do. Every skill is a `SKILL.md` file inside a directory under `.agents/skills/`. At minimum, you need two: a scheduler and an executor.

But you probably want more than the minimum. Noodle's own project ships 28 skills that cover scheduling, execution, code review, testing, reflection, and more. The fastest way to set up a good skill set is to look at what we use and adapt it.

### What to look at

The Noodle repo has two useful reference points:

**`examples/`** — starter templates. Copy one of these as your base:
- [`examples/minimal/`](https://github.com/poteto/noodle/tree/main/examples/minimal) — two skills (schedule + execute), a backlog, and a config file. The smallest working setup.
- [`examples/multi-skill/`](https://github.com/poteto/noodle/tree/main/examples/multi-skill) — adds test and deploy skills with multi-stage pipelines and routing tag overrides.

**`.agents/skills/`** — the full skill set Noodle uses to develop itself. Browse these for ideas and real-world patterns:
- [`schedule`](https://github.com/poteto/noodle/tree/main/.agents/skills/schedule) and [`execute`](https://github.com/poteto/noodle/tree/main/.agents/skills/execute) — the core loop
- [`reflect`](https://github.com/poteto/noodle/tree/main/.agents/skills/reflect), [`meditate`](https://github.com/poteto/noodle/tree/main/.agents/skills/meditate), [`ruminate`](https://github.com/poteto/noodle/tree/main/.agents/skills/ruminate) — the self-learning loop
- [`quality`](https://github.com/poteto/noodle/tree/main/.agents/skills/quality) — post-execution review gate
- [`plan`](https://github.com/poteto/noodle/tree/main/.agents/skills/plan) — systematic planning for complex tasks
- [`testing`](https://github.com/poteto/noodle/tree/main/.agents/skills/testing) — test-driven development workflow
- [`commit`](https://github.com/poteto/noodle/tree/main/.agents/skills/commit) — conventional commit messages
- [`review`](https://github.com/poteto/noodle/tree/main/.agents/skills/review) — code review walkthrough
- [`debugging`](https://github.com/poteto/noodle/tree/main/.agents/skills/debugging) — root-cause debugging methodology

### How to adapt them

Don't copy skills verbatim. Read the SKILL.md, understand what it does, then rewrite it for your human's project. Things to tailor:

- **Language and tooling.** The Noodle project is Go, so its testing skill runs `go test`. Your project might need `pytest`, `cargo test`, or `pnpm test`. Rewrite the steps.
- **Workflow preferences.** The execute skill specifies conventional commits and worktree isolation. Your human might have different commit conventions or branching strategies. Ask them.
- **Schedule triggers.** The `schedule` field is natural language. Write triggers that match how this project actually works. "After each cook session completes" is generic — "After a Go package is modified and tests haven't run yet" is specific.
- **Principles.** Read `brain/principles.md` if it exists. Those are the project's rules. Make sure your skills respect them.

### Recommended setup by project type

**Any project** — start with these:

| Skill | Purpose | Task type? |
|-------|---------|------------|
| `schedule` | Reads backlog, writes work orders | Yes |
| `execute` | Implements tasks, commits changes | Yes |
| `commit` | Conventional commit formatting | No |

**Projects that want quality gates** — add:

| Skill | Purpose | Task type? |
|-------|---------|------------|
| `quality` | Reviews completed work before merge | Yes |
| `testing` | Runs test suite against changes | Yes |
| `review` | Code review walkthrough | No |

**Projects that want self-improvement** — add:

| Skill | Purpose | Task type? |
|-------|---------|------------|
| `reflect` | Writes session learnings to the brain | Yes |
| `meditate` | Distills principles from accumulated learnings | Yes |

**Projects with complex tasks** — add:

| Skill | Purpose | Task type? |
|-------|---------|------------|
| `plan` | Breaks down large tasks into phased plans | No |
| `debugging` | Systematic root-cause analysis | No |

Task-type skills (those with a `schedule` field) run autonomously in the loop. General skills get invoked directly by agents during execution.

## Configure routing

Edit `.noodle.toml` to set the default model and any tag overrides:

```toml
mode = "auto"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
```

If the project uses multiple models for different tasks:

```toml
[routing.tags.fast]
provider = "claude"
model = "claude-sonnet-4-6"

[routing.tags.review]
provider = "claude"
model = "claude-opus-4-6"
```

The scheduling agent can reference these tags when creating orders. See [Configuration](/reference/configuration) for all options.

## Add backlog items

Edit `brain/todos.md`:

```markdown
# Todos

<!-- next-id: 2 -->

## Backlog

1. [ ] Your first task here
```

The scheduler reads this file to decide what to work on. Describe tasks clearly — the agent implementing them reads only what you write here plus the skill instructions.

## Start the loop

```sh
noodle start
```

Noodle discovers skills, runs the scheduler, and begins dispatching work. The web UI opens at `localhost:3000`.

Run `noodle status` to check on things from the terminal.

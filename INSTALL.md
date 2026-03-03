# Install Noodle

Noodle is a skill-based agent orchestration framework. You write skills — markdown files that describe what an agent should do — and Noodle's loop schedules and runs them autonomously. Think of it as a kitchen brigade: a head chef (the scheduler) reads the backlog, writes orders, and dispatches line cooks (agents) to do the work in isolated worktrees.

## How the Loop Works

```
backlog (todos.md)
  -> schedule: read project state, write orders
  -> execute: agents implement changes in worktrees
  -> merge: completed work lands on main
```

That's the minimum loop. You can add more stages — quality review, reflection, whatever your project needs — by writing skills for them. The scheduler decides when to run each one.

The scheduler reads `.noodle/mise.json` (project state: backlog, active agents, history, registered skills) and writes `.noodle/orders-next.json`. The loop promotes orders and spawns agent sessions. Each agent gets a skill loaded as its instructions. When the agent finishes, its worktree merges back.

## What Skills Are

A skill is a directory with a `SKILL.md`. The markdown body is the agent's instructions. YAML frontmatter is metadata:

```yaml
---
name: deploy
description: Deploy verified changes to staging.
schedule: "After execute completes and all tests pass."
---
```

**General skills** (no `schedule` field) are invoked directly by agents — things like `commit` or `debugging`. **Scheduled skills** (with `schedule`) run autonomously. The scheduler reads the `schedule` value as prose and decides when conditions are met.

A working loop needs at minimum: a **schedule skill** (reads backlog, writes orders) and one **task-type skill** like **execute** (does the work). Additional stages (review, deploy, reflect — whatever fits your workflow) make the loop better but aren't required to start.

---

## Setup

Work through these steps in order. Each has a skip condition — if the check passes, move to the next step.

### 1. Install the binary

**Skip if:** `noodle --version` succeeds.

Detect the platform and install:

- **macOS:** `brew install poteto/tap/noodle`
- **Linux/Windows:** Download the binary for your platform from `https://github.com/poteto/noodle/releases/latest` and place it on PATH.

Verify: `noodle --version` prints a version string.

Save the installed version (e.g., `v0.1.3`) — you'll need it for the next step.

### 2. Install the noodle skill

**Skip if:** `.agents/skills/noodle/SKILL.md` AND all files in `.agents/skills/noodle/references/` exist.

If only some files exist (interrupted install), fetch the missing ones.

The noodle skill is a reference manual that teaches agents how to use the CLI and write skills. It's the one skill you copy verbatim — everything else you'll write yourself.

Fetch these files from GitHub, using the installed version as the tag (fall back to `main` if the version is unknown):

```
.agents/skills/noodle/SKILL.md
.agents/skills/noodle/references/skill-authoring.md
.agents/skills/noodle/references/configuration.md
```

The raw URLs follow this pattern:
```
https://raw.githubusercontent.com/poteto/noodle/{version}/.agents/skills/noodle/SKILL.md
https://raw.githubusercontent.com/poteto/noodle/{version}/.agents/skills/noodle/references/skill-authoring.md
https://raw.githubusercontent.com/poteto/noodle/{version}/.agents/skills/noodle/references/configuration.md
```

Replace `{version}` with the tag from step 1 (e.g., `v0.1.3`).

Create the directories if they don't exist. Write the fetched content to the corresponding paths.

### 3. Configure

**Skip if:** `.noodle.toml` exists at the project root.

Ask the user which provider they use: **Claude** or **Codex**. Then generate `.noodle.toml`:

For **Claude**:
```toml
mode = "supervised"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
```

For **Codex**:
```toml
mode = "supervised"

[routing.defaults]
provider = "codex"
model = "gpt-5.3-codex"

[skills]
paths = [".agents/skills"]
```

This is a minimal config. The user can expand it later — see [the configuration reference](https://poteto.github.io/noodle/reference/configuration) for all options.

### 4. Add .noodle/ to .gitignore

**Skip if:** `.gitignore` already contains `.noodle/`.

Grep `.gitignore` for `.noodle/`. If absent (or `.gitignore` doesn't exist), append:

```
# Noodle runtime state
.noodle/
```

### 5. Seed the backlog

**Skip if:** `todos.md` exists at the project root.

Create `todos.md` with these bootstrap tasks:

```markdown
# Backlog

- [ ] Write a schedule skill and execute skill for this project
```

This is the real onboarding task. When the loop picks it up, the agent will read the project (past conversation logs in `.claude/` or `.codex/`, existing code, language, tools) and write skills adapted to how the project actually works. See "Writing Skills" below for guidance.

If `todos.md` already exists but `.agents/skills/schedule/` doesn't, skip the file write but continue to step 6 — the user still needs skills.

### 6. Ask about brainmaxxing

Ask the user: "Do you want brainmaxxing? It adds a `brain/` vault for persistent memory across sessions, plus reflect, meditate, and ruminate skills."

If yes, follow the install instructions at `https://github.com/poteto/brainmaxxing`.

If no, move on.

### 7. First run

Tell the user to start the loop:

```
noodle start
```

Noodle will scaffold `.noodle/` runtime state, the scheduler will read the backlog, and the loop begins. The first order will be the bootstrap task from step 5 — writing skills for this project.

---

## Writing Skills

When the loop picks up the "write skills" task from the backlog, you need to write two skills: a **schedule skill** and an **execute skill**. Use the noodle skill's reference docs (installed in step 2) for the orders schema and CLI details. Browse Noodle's own skills at `https://github.com/poteto/noodle/tree/main/.agents/skills/` for patterns — but adapt them to this project, don't copy them.

### Schedule Skill

The schedule skill reads project state and writes orders. It needs to:

- **Read the backlog** from `.noodle/mise.json` (which syncs from `todos.md` or whatever backlog adapter is configured)
- **Write `orders-next.json`** — never `orders.json` directly. The loop atomically promotes orders. Use `noodle schema orders` to get the schema.
- **Route to providers/models** based on `routing.defaults` in `.noodle.toml` and the task's complexity
- **Have a `schedule` frontmatter field** — something like "when orders are empty, after backlog changes, or when session history suggests re-evaluation"

### Execute Skill

The execute skill does the actual work. It needs to:

- **Scope and decompose** the assigned task into discrete changes
- **Work in worktrees** — never on main. Use `noodle worktree create/merge` commands.
- **Verify** — run whatever build, test, and lint commands this project uses
- **Commit** — follow this project's commit conventions
- **Have a `schedule` frontmatter field** — something like "when backlog items are ready for implementation"

### Key Principle

Read the project first. Look at existing code, test commands, CI config, past conversation logs. Write skills that match how this project actually works — its language, test runner, CI, and workflow.

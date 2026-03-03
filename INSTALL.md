# Install Noodle

Noodle is a skill-based agent orchestration framework. You write skills — markdown files that describe what an agent should do — and Noodle's loop schedules and runs them autonomously. Think of it as a kitchen brigade: a head chef (the scheduler) reads the backlog, writes orders, and dispatches line cooks (agents) to do the work in isolated worktrees.

## How the Loop Works

```
backlog (todos.md)
  -> schedule: read project state, write orders
  -> execute: agents implement changes in worktrees
  -> quality: cross-provider review of the work
  -> reflect: capture learnings for next cycle
  -> merge: completed work lands on main
```

The scheduler reads `.noodle/mise.json` (project state: backlog, active agents, history, registered skills) and writes `.noodle/orders-next.json`. The loop promotes orders and spawns agent sessions. Each agent gets a skill loaded as its instructions. When the agent finishes, its worktree merges back.

## What Skills Are

A skill is a directory with a `SKILL.md`. The markdown body is the agent's instructions. YAML frontmatter is metadata:

```yaml
---
name: quality
description: Post-execution quality gate.
schedule: "Follow-up stage after execute. Cross-provider review preferred."
---
```

**General skills** (no `schedule` field) are invoked directly by agents — things like `commit` or `debugging`. **Scheduled skills** (with `schedule`) run autonomously. The scheduler reads the `schedule` value as prose and decides when conditions are met.

A working loop needs at minimum: a **schedule skill** (reads backlog, writes orders) and one **task-type skill** like **execute** (does the work). Quality and reflect skills make the loop better but aren't required to start.

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
- [ ] Browse github.com/poteto/noodle/.agents/skills/ and find skills that might be useful to adapt to this codebase
```

The first task is the real onboarding. When the loop picks it up, the agent will read the user's project (past conversation logs in `.claude/` or `.codex/`, existing code, language, tools) and write skills adapted to how the user actually works. The second task broadens the skill set.

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

## Writing Skills: Reference Implementations

When the loop picks up the "write skills" task from the backlog, use Noodle's own skills as reference. Read them, understand the patterns, then write skills that fit the user's project. Do not copy them verbatim — every project has different languages, tools, test commands, and workflows.

### Schedule Skill

Reference: `https://raw.githubusercontent.com/poteto/noodle/v0.1.3/.agents/skills/schedule/SKILL.md`

What matters:

- **The `schedule` frontmatter field** — describes when the scheduler should re-evaluate. For the schedule skill itself, this is "when orders are empty, after backlog changes, or when session history suggests re-evaluation."
- **Reading the backlog** — the scheduler reads from `.noodle/mise.json`, which contains the synced backlog. The backlog source is usually `todos.md` but could be any adapter.
- **Writing `orders-next.json`** — the scheduler writes orders here; the loop atomically promotes them. Never write `orders.json` directly. Use `noodle schema orders` to get the schema.
- **Model routing** — the schedule skill routes stages to providers/models based on the user's config and the task's complexity. Read `routing.defaults` from `.noodle.toml` for the user's preferred provider.

The schedule skill also has reference files:

```
.agents/skills/schedule/references/examples.md   — order JSON examples
.agents/skills/schedule/references/events.md      — event type catalog
```

### Execute Skill

Reference: `https://raw.githubusercontent.com/poteto/noodle/v0.1.3/.agents/skills/execute/SKILL.md`

What matters:

- **The `schedule` frontmatter field** — describes when this task type runs. For execute, it's "when backlog items with linked plans are ready for implementation."
- **Verification strategy** — this is project-specific. Noodle's execute skill runs `go test`, `go vet`, and `pnpm test:smoke`. Your execute skill should run whatever build, test, and lint commands the user's project uses.
- **Worktree workflow** — agents work in isolated worktrees, never on main. The execute skill teaches this process.
- **Commit conventions** — conventional commits (`feat`, `fix`, `refactor`, etc.) with scope. Adapt to whatever the user's project already uses.

### Key Principle

The agent adapts, not copies. Read the reference implementations to understand what a schedule skill needs to do (read state, write orders, route models) and what an execute skill needs to do (scope, implement, verify, commit). Then write skills that fit the user's project — their language, their test runner, their CI, their workflow.

# Install Noodle

Noodle is a skill-based agent orchestration framework. You write skills (markdown files describing what an agent should do) and Noodle's loop schedules and runs them. Think kitchen brigade: a head chef (the scheduler) reads the backlog, writes orders, and dispatches line cooks (agents) to do the work in isolated worktrees.

## How the loop works

The scheduler reads `.noodle/mise.json` (project state: backlog, active agents, history, registered skills) and writes `.noodle/orders-next.json`. The loop promotes orders and spawns agent sessions. Each agent gets a skill loaded as its instructions. When the agent finishes, its worktree merges back.

The minimum loop is schedule then execute. Noodle handles merging completed work back to main. You can add more stages (quality review, reflection, etc.) by writing skills for them.

## The `schedule` frontmatter field

Noodle adds one thing to skills: a `schedule` field in the YAML frontmatter. Skills without it work normally (agents invoke them directly). Skills with it become autonomous. The scheduler reads the `schedule` value as prose and decides when conditions are met.

```yaml
schedule: "After execute completes and all tests pass."
```

A working loop needs at least two scheduled skills: a schedule skill (reads backlog, writes orders) and one task-type skill like execute (does the work).

---

## Setup

Work through these steps in order. Each has a skip condition. If the check passes, move on.

### 1. Install the binary

**Skip if:** `noodle --version` succeeds.

Detect the platform and install:

- **macOS:** `brew install poteto/tap/noodle`
- **Linux/Windows:** Download the binary for your platform from `https://github.com/poteto/noodle/releases/latest` and place it on PATH.

Verify: `noodle --version` prints a version string.

### 2. Install the noodle skill

**Skip if:** `.agents/skills/noodle/SKILL.md` AND all files in `.agents/skills/noodle/references/` exist.

If only some files exist (interrupted install), fetch the missing ones.

The noodle skill teaches agents how to use the CLI and write skills. Copy this one verbatim. Everything else you write yourself.

Fetch these files from GitHub:

```
https://raw.githubusercontent.com/poteto/noodle/main/.agents/skills/noodle/SKILL.md
https://raw.githubusercontent.com/poteto/noodle/main/.agents/skills/noodle/references/skill-authoring.md
https://raw.githubusercontent.com/poteto/noodle/main/.agents/skills/noodle/references/configuration.md
```

Write them to the same paths under the project root (`.agents/skills/noodle/...`). Create directories as needed.

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

Minimal config. The user can expand it later. See [the configuration reference](https://poteto.github.io/noodle/reference/configuration) for all options.

### 4. Add .noodle/ to .gitignore

**Skip if:** `.gitignore` already contains `.noodle/`.

Grep `.gitignore` for `.noodle/`. If absent (or `.gitignore` doesn't exist), append:

```
# Noodle runtime state
.noodle/
```

### 5. Write schedule and execute skills

**Skip if:** `.agents/skills/schedule/SKILL.md` AND `.agents/skills/execute/SKILL.md` both exist.

Start by understanding how your human works. Read the conversation history in `~/.claude/` and `~/.codex/` for this project. Look at what commands they run, how they test, how they review code, what they ask agents to do repeatedly. This is the ground truth for what the skills should encode. Existing code, CI config, and test commands fill in the rest.

Then browse `https://github.com/poteto/noodle/tree/main/.agents/skills/` for patterns. Use Noodle's own schedule and execute skills as starting points, but adapt them to this project. Don't copy. The noodle skill's reference docs (from step 2) cover the orders schema and CLI details.

**Schedule skill** (`schedule/SKILL.md`): Reads project state and writes orders. Reads the backlog from `.noodle/mise.json` (synced from `todos.md` or the configured backlog adapter). Writes `orders-next.json`, not `orders.json` directly. Routes to providers/models based on `routing.defaults` in `.noodle.toml`. Include a `schedule` frontmatter field, something like "when orders are empty, after backlog changes, or when session history suggests re-evaluation."

**Execute skill** (`execute/SKILL.md`): Does the work. Scopes and decomposes assigned tasks into discrete changes. Works in worktrees, never on main (`noodle worktree create/merge`). Verifies with whatever build, test, and lint commands this project uses. Commits following this project's conventions. Include a `schedule` frontmatter field, something like "when backlog items are ready for implementation."

### 6. Seed the backlog

**Skip if:** `todos.md` exists at the project root.

Create an empty `todos.md`:

```markdown
# Backlog
```

Ask the user what they want to work on and add it as the first task.

### 7. Ask about brainmaxxing

Ask the user: "Do you want brainmaxxing? It adds a `brain/` vault for persistent memory across sessions, plus reflect, meditate, and ruminate skills."

If yes, follow the install instructions at `https://github.com/poteto/brainmaxxing`.

If no, move on.

### 8. First run

Tell the user to start the loop:

```
noodle start
```

Noodle scaffolds `.noodle/` runtime state, the scheduler reads the backlog, and the loop begins.

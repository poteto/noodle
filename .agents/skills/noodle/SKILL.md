---
name: noodle
description: >-
  Operate the Noodle CLI — explain commands, find flags, create/edit .noodle.toml config.
---

# Noodle

Noodle is an open-source AI coding framework. Skills are the only extension point. An LLM schedules work, Go code executes it mechanically. Everything is a file — queue.json, mise.json, verdicts, control.ndjson. No hidden state.

Kitchen brigade model: the human is the Chef (strategy and judgment), Prioritize is the Sous Chef (scheduling), Cook does the work, Quality reviews it.

## Config Reference

Noodle reads `.noodle.toml` at project root. If missing, `noodle start` scaffolds a minimal one on first run.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `adapters` | table | {backlog} | Adapter configs keyed by adapter name (e.g. backlog) |
| `agents.claude.args` | array | [] | Extra CLI arguments for Claude Code |
| `agents.claude.path` | string | "" | Custom path to Claude Code binary |
| `agents.codex.args` | array | [] | Extra CLI arguments for Codex CLI |
| `agents.codex.path` | string | "" | Custom path to Codex CLI binary |
| `autonomy` | string | "review" | How much human oversight the loop requires: full, review, or approve |
| `concurrency.max_cooks` | integer | 4 | Maximum concurrent cook sessions |
| `monitor.poll_interval` | string | "5s" | How often the monitor checks session status |
| `monitor.stuck_threshold` | string | "120s" | Duration before a cook is considered stuck |
| `monitor.ticket_stale` | string | "30m" | Duration before a ticket is considered stale |
| `phases` | table | {debugging, oops} | Map of phase names to skill names for lifecycle hooks |
| `plans.on_done` | string | "keep" | What to do with completed plans: keep or remove |
| `prioritize.model` | string | "claude-sonnet" | Model used for scheduling sessions |
| `prioritize.run` | string | "after-each" | When to run scheduling: after-each, after-n, or manual |
| `prioritize.skill` | string | "prioritize" | Skill name loaded for scheduling sessions |
| `recovery.max_retries` | integer | 3 | Maximum retry attempts for failed cooks |
| `recovery.retry_suffix_pattern` | string | "-recover-%d" | Naming pattern for retry sessions (must include %d) |
| `routing.defaults.model` | string | "claude-opus-4-6" | Default model name for cook sessions |
| `routing.defaults.provider` | string | "claude" | Default LLM provider for cook sessions (claude or codex) |
| `routing.tags` | table | {} | Per-tag model overrides keyed by tag name |
| `runtime.cursor.api_key_env` | string | "" |  |
| `runtime.cursor.base_url` | string | "" |  |
| `runtime.cursor.repository` | string | "" |  |
| `runtime.default` | string | "tmux" | Default runtime command template for spawning cooks |
| `runtime.sprites.base_url` | string | "" |  |
| `runtime.sprites.git_token_env` | string | "" |  |
| `runtime.sprites.sprite_name` | string | "" |  |
| `runtime.sprites.token_env` | string | "" |  |
| `skills.paths` | array | [".agents/skills"] | Ordered search paths for skill resolution |

### Minimal config

```toml
autonomy = "review"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
```

### Adapter config

Adapters bridge your backlog or plan system to Noodle. Each adapter has a skill (teaches agents the semantics) and scripts (deterministic commands for CRUD actions).

```toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
add = ".noodle/adapters/backlog-add"
done = ".noodle/adapters/backlog-done"
edit = ".noodle/adapters/backlog-edit"
```

Scripts can be any executable — shell scripts, binaries, or inline commands like `gh issue close`. Noodle calls them mechanically; the skill teaches agents when and why to use them.

### Routing tags

Override the default model for specific task categories:

```toml
[routing.tags.frontend]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.backend]
provider = "codex"
model = "gpt-5.3-codex"
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `noodle start` | Run the scheduling loop |
| `noodle status` | Show compact runtime status |
| `noodle debug` | Dump canonical runtime debug state |
| `noodle skills` | List resolved skills |
| `noodle skills list` | List all resolved skills |
| `noodle schema` | Print generated schema docs for Noodle runtime contracts |
| `noodle schema list` | List available schema targets |
| `noodle worktree` | Manage linked git worktrees |
| `noodle worktree create` | Create a new linked worktree |
| `noodle worktree exec` | Run command inside worktree (CWD-safe) |
| `noodle worktree merge` | Merge a worktree branch back to main |
| `noodle worktree cleanup` | Remove a worktree without merging |
| `noodle worktree list` | List all worktrees with merge status |
| `noodle worktree prune` | Remove merged and patch-equivalent worktrees |
| `noodle worktree hook` | Run worktree session hook |
| `noodle plan` | Manage plans (create, done, phase-add, list) |
| `noodle plan create` | Create a plan from a todo |
| `noodle plan activate` | Mark a plan as active |
| `noodle plan done` | Mark a plan as done |
| `noodle plan phase-add` | Add a phase to a plan |
| `noodle plan list` | List all plans |
| `noodle stamp` | Stamp NDJSON logs and emit canonical sidecar events |
| `noodle dispatch` | Dispatch a cook session in tmux |
| `noodle mise` | Build and print the current mise brief |

### Flags

`noodle start`:
- `--once` (bool): Run one scheduling cycle and exit

`noodle worktree cleanup`:
- `--force` (bool): Remove even when unmerged commits exist

`noodle stamp`:
- `--output` (`-o`) (string): Output path for stamped NDJSON
- `--events` (string): Output path for canonical sidecar events

`noodle dispatch`:
- `--name` (string), default `cook`: Session name
- `--prompt` (string): Prompt text for the dispatched session
- `--provider` (string): Provider (claude or codex)
- `--model` (string): Model name
- `--skill` (string): Skill name to inject
- `--reasoning-level` (string): Reasoning level
- `--worktree` (string): Linked worktree path
- `--max-turns` (int): Max turns
- `--budget-cap` (float64): Budget cap
- `--env` ([]string): Extra env vars (KEY=VALUE)


## Adapter Setup

Adapters are optional. If you omit `[adapters.plans]` from config, Noodle runs with backlog only. If both adapters are omitted, the mise contains only internal state.

### Writing adapter scripts

Each adapter action (sync, add, done, edit) maps to a script path in config. Scripts receive arguments via environment variables and must produce NDJSON output for sync actions.

1. **Sync** — reads all items from your system, writes NDJSON to stdout. Each line is a `BacklogItem` or `PlanItem`.
2. **Add** — creates a new item. Receives `NOODLE_TITLE` and `NOODLE_BODY` env vars.
3. **Done** — marks an item complete. Receives `NOODLE_ID`.
4. **Edit** — updates an item. Receives `NOODLE_ID`, `NOODLE_FIELD`, `NOODLE_VALUE`.

### Markdown backlog (default)

The default adapter reads `brain/todos.md` — a markdown file with numbered items. Scripts live at `.noodle/adapters/backlog-*`.

### GitHub Issues

```toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = "gh issue list --json number,title,body,labels,state"
add = "gh issue create"
done = "gh issue close"
edit = "gh issue edit"
```

### Linear

Use the Linear CLI or API. The adapter pattern is the same — write scripts that call the Linear API and output NDJSON.

## Hook Installation

### Brain injection hook

Injects brain vault content into the agent's context at session start. Add to `.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "SessionStart",
        "hooks": [
          {
            "type": "command",
            "command": "noodle worktree hook"
          }
        ]
      }
    ]
  }
}
```

## Skill Management

Skills live in `.agents/skills/` by default. Each skill is a directory with a `SKILL.md` file and optional `references/` subdirectory.

### Search path precedence

Paths in `skills.paths` are searched in order. First match wins. Project skills override user-level skills of the same name.

### Installing a skill

Copy the skill directory to your first skill path:

```sh
cp -r /path/to/skill .agents/skills/my-skill
```

### Task-type skills

Skills with a `task_type` in their frontmatter are discovered as task types by the scheduling loop. These are loaded automatically for their respective session types (prioritize, review, etc.).

## Troubleshooting

### `noodle debug`

Dumps the full runtime state: config validation, active sessions, queue, mise, and diagnostics. Run this first when something is wrong.

### Common issues

1. **"tmux is not available on PATH"** — Install tmux. Noodle uses tmux to spawn and manage cook sessions.
2. **"fatal config diagnostics prevent start"** — Run `noodle debug` to see which config fields are invalid. Fix them in `.noodle.toml`.
3. **Missing adapter scripts** — `noodle start` reports missing script paths as repairable diagnostics. Create the scripts or update the paths in config.
4. **Stale worktrees** — Run `noodle worktree list` to check status, then `noodle worktree prune` to clean up merged branches.

### Config validation

`noodle start` validates config on every run. Diagnostics are classified as:
- **Fatal** — blocks startup (missing tmux, invalid routing defaults)
- **Repairable** — warns but allows startup (missing adapter scripts)

On interactive terminals, `noodle start` offers to spawn a repair session for repairable issues.

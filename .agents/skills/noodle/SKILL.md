---
name: noodle
description: >-
  Operate the Noodle CLI — explain commands, find flags, create/edit .noodle.toml config.
---

# Noodle

Noodle is an open-source AI coding framework. Skills are the only extension point. An LLM schedules work, Go code executes it mechanically. Everything is a file — queue-next.json, mise.json, control.ndjson. No hidden state.

## Task-Type Skill Frontmatter

Skills with a `noodle:` block in their YAML frontmatter are discovered as task types by the scheduling loop. The prioritize skill reads `task_types[].schedule` from mise to decide when to schedule each type.

```yaml
---
name: my-task-type
description: What this task type does
noodle:
  schedule: "When to schedule this task type"
  permissions:
    merge: true
---
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `noodle.schedule` | yes | — | Hint for the prioritize skill on when to schedule this type |
| `noodle.permissions.merge` | no | `true` | Auto-merge worktree on success. Set `false` to park for human approval |

When `permissions.merge` is `false`, the loop parks the completed worktree instead of auto-merging. The human reviews and approves via the TUI Reviews tab.

The global `autonomy` config (`auto` or `approve`) overrides per-skill merge permissions: `approve` mode parks all worktrees regardless of the skill's `permissions.merge` value.

## Config Reference

Noodle reads `.noodle.toml` at project root. If missing, `noodle start` scaffolds a minimal one on first run.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `adapters` | table | {backlog} | Adapter configs keyed by adapter name (e.g. backlog) |
| `agents.claude.args` | array | [] | Extra CLI arguments for Claude Code |
| `agents.claude.path` | string | "" | Custom path to Claude Code binary |
| `agents.codex.args` | array | [] | Extra CLI arguments for Codex CLI |
| `agents.codex.path` | string | "" | Custom path to Codex CLI binary |
| `autonomy` | string | "auto" | How much human oversight the loop requires: auto or approve |
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
autonomy = "auto"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
```

For adapter config and routing tags, see [references/adapters.md](references/adapters.md) and [references/config-schema.md](references/config-schema.md).

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


## Skill Management

Skills live in `.agents/skills/` by default. Paths in `skills.paths` are searched in order; first match wins. Install a skill by copying its directory to your skill path.

## Troubleshooting

Run `noodle debug` to dump the full runtime state. Common issues:

1. **"tmux is not available on PATH"** — Install tmux.
2. **"fatal config diagnostics prevent start"** — Run `noodle debug`, fix fields in `.noodle.toml`.
3. **Missing adapter scripts** — Create scripts or update paths in config.
4. **Stale worktrees** — `noodle worktree list`, then `noodle worktree prune`.

## References

- [references/config-schema.md](references/config-schema.md) — routing tags, config validation
- [references/adapters.md](references/adapters.md) — adapter setup, script writing, provider examples
- [references/hooks.md](references/hooks.md) — brain injection hook, settings.json setup

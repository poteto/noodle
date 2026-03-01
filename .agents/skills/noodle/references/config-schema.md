# Config Schema Reference

## Run mode

The `mode` config field sets the run mode governing loop behavior:

```toml
mode = "auto"        # full automation: schedule, dispatch, retry, merge
# mode = "supervised" # human approves merges and retries
# mode = "manual"     # human triggers everything
```

### Before/after: mode semantics

**Before (V1):** `autonomy` accepted `auto` or `approve`. `approve` parked worktrees for human merge approval.

**After (V2):** `mode` accepts `auto`, `supervised`, or `manual`. Each mode gates four actions independently:

| Mode | Schedule | Dispatch | Auto-retry | Auto-merge |
|------|----------|----------|------------|------------|
| `auto` | yes | yes | yes | yes |
| `supervised` | yes | yes | no | no |
| `manual` | no | no | no | no |

Mode transitions are epoch-stamped. In-flight effects created under a previous epoch are cancelled, not applied.

## Routing tags

Override the default model for specific task categories:

```toml
[routing.tags.frontend]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.backend]
provider = "codex"
model = "gpt-5.3-codex"
```

## Runtime configuration

The `runtime` section controls dispatch backends:

```toml
[runtime]
default = "process"  # process | sprites | cursor

[runtime.process]
max_concurrent = 4

[runtime.sprites]
token_env = "SPRITES_TOKEN"
max_concurrent = 50

[runtime.cursor]
api_key_env = "CURSOR_API_KEY"
max_concurrent = 10
```

Each runtime declares capabilities: `steerable`, `polling`, `remote_sync`, `heartbeat`. The dispatcher queries capabilities to decide polling intervals, steering support, and sync behavior rather than branching on runtime names.

## Config validation

`noodle start` validates config on every run. Diagnostics are classified as:
- **Fatal** -- blocks startup (invalid routing defaults, unknown runtime default)
- **Repairable** -- warns but allows startup (missing adapter scripts)

On interactive terminals, `noodle start` offers to spawn a repair session for repairable issues.

## Canonical state files

The loop maintains these files in `.noodle/`:

| File | Description |
|------|-------------|
| `orders.json` | Projected orders with stage lifecycle status |
| `orders-next.json` | Scheduler output; promoted atomically by the loop |
| `status.json` | Runtime status (active order IDs, loop state, run mode, max cooks) |
| `state.json` | Schema version marker |
| `control.ndjson` | Control command input (append-only) |
| `control-ack.ndjson` | Control command acknowledgments |
| `mise.json` | Mise brief for the scheduler |

### Before/after: control commands

**Before (V1):** The `autonomy` control action toggled between `auto` and `approve`.

**After (V2):** The `mode` control action accepts `auto`, `supervised`, or `manual`. Mode transitions are tracked with a monotonic epoch. New control actions `advance`, `add-stage`, and `park-review` support fine-grained order management.

### Before/after: file contracts

**Before (V1):** `status.json` contained `autonomy` with values `auto` or `approve`. No projection versioning.

**After (V2):** `status.json` uses the `mode` field with values `auto`, `supervised`, or `manual`. Projection files (`orders.json`, `state.json`) are written atomically by the projection layer with deterministic hashing and versioning. The snapshot API includes `mode`, `mode_epoch`, `schema_version`, and `last_event_id`.

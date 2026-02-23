Back to [[plans/27-remote-dispatchers/overview]]

# Phase 8: Config, schemas, and prioritize skill

**Routing:** `claude` / `claude-opus-4-6` — judgment needed for schema design and skill update

## Goal

Three things: (1) user-level config for available runtimes, (2) surface available runtimes in mise.json so the prioritize agent can see them, (3) update the prioritize skill to make runtime routing decisions, (4) add `runtime` field to queue.json schema, (5) relax provider validation for remote runtimes.

## Data structures

- `SpritesConfig` struct — env var name for token, base URL, default sprite name
- `CursorConfig` struct — env var name for API key, base URL, default repo URL
- Extend `RuntimeConfig` with `Sprites SpritesConfig` and `Cursor CursorConfig`
- `available_runtimes` array in mise.json routing section
- `runtime` string field on queue.json items

## Changes

**`config/config.go`**
Add `SpritesConfig` and `CursorConfig` structs under `RuntimeConfig`. **No secrets in config** — tokens are read from environment variables. Config stores only non-secret settings and optionally the env var name to read:
```toml
[runtime]
default = "tmux"

[runtime.sprites]
# Token read from $SPRITES_TOKEN (well-known default)
# token_env = "SPRITES_TOKEN"  # optional override if user uses a different env var
base_url = "https://api.sprites.dev/v1"
sprite_name = "noodle-worker"

[runtime.cursor]
# API key read from $CURSOR_API_KEY (well-known default)
# api_key_env = "CURSOR_API_KEY"  # optional override
base_url = "https://api.cursor.com"
repository = "https://github.com/user/repo"
```

Each config struct has a well-known default env var (`SPRITES_TOKEN`, `CURSOR_API_KEY`) and an optional `token_env`/`api_key_env` override field. The `Token()` / `APIKey()` method reads `os.Getenv` at call time — never stored in config or serialized.

Add a method `Config.AvailableRuntimes() []string` that returns `["tmux"]` plus any configured remote backends. A backend is "configured" when its config section exists **and** its token env var is set (e.g., `["tmux", "sprites"]` when `$SPRITES_TOKEN` is non-empty).

**`config/config.go`**
Relax `validateProvider()`: move hard-coded "claude"/"codex" check into `TmuxBackend` where it's meaningful (needs to resolve a binary). Accept any non-empty provider string at the config/request boundary.

**Mise schema (`cmd_mise.go` or wherever mise is generated)**
Add `routing.available_runtimes: ["tmux", "sprites"]` to mise.json output. Populated from `Config.AvailableRuntimes()`. The prioritize agent reads this to know what dispatch options exist.

**Queue schema (`loop/types.go`)**
Add `Runtime string` field to `QueueItem` (already done in Phase 2).

**`noodle schema queue` and `noodle schema mise`**
Update schema docs to include the new fields.

**`.agents/skills/prioritize/SKILL.md`**
Add a "Runtime Routing" section to the prioritize skill. The prioritize agent should:
- Read `routing.available_runtimes` from mise.json
- Default to `"tmux"` when only local is available
- Use remote runtimes when they make sense for the task (e.g., long-running execute tasks on Sprites, fire-and-forget tasks on Cursor)
- Set `"runtime": "sprites"` or `"runtime": "cursor"` on queue items
- Use `skill-creator` skill for this update

## Verification

### Static
- Compiles, passes vet
- Schema commands show new fields
- Existing config tests updated for relaxed provider validation

### Runtime
- Unit test: `AvailableRuntimes()` returns `["tmux"]` with no remote config
- Unit test: `AvailableRuntimes()` returns `["tmux", "sprites"]` when sprites config exists and `$SPRITES_TOKEN` is set
- Unit test: `AvailableRuntimes()` returns `["tmux"]` when sprites config exists but `$SPRITES_TOKEN` is empty (no token = not available)
- Unit test: `SpritesConfig.Token()` reads from custom env var when `token_env` is set
- Unit test: mise output includes `routing.available_runtimes`
- Unit test: queue item with `runtime: "sprites"` round-trips through JSON

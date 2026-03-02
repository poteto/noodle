Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 5 — Plugin Config Passthrough

## Goal

Replace hardcoded plugin-specific config types (`SpritesConfig`, `CursorConfig`) with a generic passthrough system. Any `[runtime.NAME]` TOML section for a plugin runtime is parsed and converted to JSON at the parse boundary, then forwarded to the plugin's `initialize` RPC as `json.RawMessage`.

## Changes

**Modified file: `config/types_defaults.go`**

- Add `Plugins map[string]json.RawMessage` to `RuntimeConfig` — captures unknown `[runtime.*]` sections, converted from TOML to JSON at parse time (the plugin protocol speaks JSON, not TOML)
- Keep `ProcessConfig` as a named field (process is built-in, not a plugin)
- `AvailableRuntimes()` includes names from the `Plugins` map (in addition to "process"). Note: a runtime is only *active* when both discovered AND configured (phase 4 enforces this).
- Remove `spritesDefined` / `cursorDefined` tracking — plugin availability is determined by the discovery+config intersection (phase 4)

**Modified file: `config/parse.go`**

- Parse known sections (`[runtime.process]`) into typed structs
- Capture all other `[runtime.*]` sections, convert TOML→JSON at the parse boundary, store as `json.RawMessage` in the `Plugins` map
- Pass JSON config bytes to `PluginHost` during initialization (phase 2's `NewPluginHost`)

**Modified file: `config/diagnostics.go`**

- Remove Sprites-specific validation
- Generic plugin config validation: warn if a `[runtime.NAME]` section exists but no matching plugin is discovered

**Modified file: `config/config_test.go`**

- Update tests to reflect generic passthrough
- Test unknown `[runtime.myvm]` section captured in `Plugins` map
- Test process config still parsed into typed struct

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Config architecture redesign — judgment on backward compat, TOML parsing edge cases |

## Verification

### Static
- `go build ./config/...`
- `go vet ./config/...`
- Existing config tests updated and passing

### Runtime
- Parse a `.noodle.toml` with `[runtime.sprites]` section → verify JSON bytes captured in `Plugins["sprites"]` (not TOML)
- Parse a `.noodle.toml` with `[runtime.myvm]` custom section → verify TOML→JSON conversion preserves all fields
- Parse without any plugin sections → verify empty `Plugins` map, no errors
- Diagnostics warn on `[runtime.sprites]` without a discovered sprites plugin

Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 4 — Plugin Discovery

## Goal

Automatically discover installed runtime plugins and register them in the runtime map alongside the built-in process runtime.

## Changes

**New file: `plugin/discover.go`**

`Discover() ([]PluginInfo, error)` — scans for `noodle-runtime-*` binaries in:
1. `~/.noodle/plugins/` (user-installed)
2. Adjacent to the `noodle` binary (bundled/co-installed)
3. `$PATH` (system-wide)

Returns deduplicated list (first match wins for a given name). Each `PluginInfo` contains the binary path and the extracted runtime name (strip `noodle-runtime-` prefix).

**Reserved names:** Discovery must skip plugins whose extracted name matches a built-in runtime (`process`). Prevents PATH hijacking of the universal fallback.

**Modified file: `loop/defaults.go`**

Replace the hardcoded Sprites wiring block with a generic plugin loop:
1. Call `plugin.Discover()`
2. For each discovered plugin that also has a matching `[runtime.NAME]` config section (availability rule: discovered AND configured), create a `PluginRuntime` via `NewPluginHost()` with a **5-second initialize timeout**
3. If a plugin fails `initialize` (error or timeout), log a warning, skip it, continue with remaining plugins. Never abort startup for a plugin failure.
4. Add successful plugins to the `runtimes` map
5. Register capabilities from the plugin's `initialize` response in the `rtcap.Registry`

Process runtime remains hardcoded (it's the kernel). Plugin runtimes are additive.

**New file: `plugin/discover_test.go`**
- Test discovery with mock binaries in a temp directory
- Test dedup (same name in multiple paths → first wins)
- Test non-executable files are skipped
- Test empty directories
- Test reserved name "process" is skipped
- Test plugin that fails initialize is skipped with warning (doesn't abort startup)
- Test plugin that hangs initialize is skipped after 5s timeout
- Test availability rule: discovered plugin without config section is not registered

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.3-codex` | Straightforward path scanning and registration |

## Verification

### Static
- `go build ./plugin/... ./loop/...`
- `go vet ./...`

### Runtime
- Create temp dirs with mock `noodle-runtime-test` binaries, verify discovery finds them
- Verify precedence: `~/.noodle/plugins/` > adjacent > `$PATH`
- Verify `loop/defaults.go` registers discovered plugins in the runtime map
- Verify startup completes even when one plugin fails initialize
- Verify `noodle-runtime-process` on PATH does not override built-in
- Integration test: discovered plugin runtime responds to dispatch

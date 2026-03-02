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

**Modified file: `loop/defaults.go`**

Replace the hardcoded Sprites wiring block with a generic plugin loop:
1. Call `plugin.Discover()`
2. For each discovered plugin, create a `PluginRuntime` via `NewPluginHost()`
3. Add to the `runtimes` map
4. Register capabilities from the plugin's `initialize` response in the `rtcap.Registry`

Process runtime remains hardcoded (it's the kernel). Plugin runtimes are additive.

**New file: `plugin/discover_test.go`**
- Test discovery with mock binaries in a temp directory
- Test dedup (same name in multiple paths → first wins)
- Test non-executable files are skipped
- Test empty directories

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
- Integration test: discovered plugin runtime responds to dispatch

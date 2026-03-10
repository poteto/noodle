Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 9 — Remove Sprites from Core

## Goal

Delete all Sprites-specific code from the Noodle core binary. After this phase, Noodle's core has no knowledge of Sprites — it's just another plugin.

## Changes

**Delete files:**
- `dispatcher/sprites_dispatcher.go`
- `dispatcher/sprites_session.go`

**Modified file: `config/types_defaults.go`**
- Remove `SpritesConfig` struct
- Remove `Sprites` field from `RuntimeConfig`
- Remove `spritesDefined` tracking
- Remove Sprites from `AvailableRuntimes()` hardcoded list (plugins handle this now)

**Modified file: `config/parse.go`**
- Remove Sprites-specific config parsing
- `[runtime.sprites]` now handled by generic plugin passthrough (phase 5)

**Modified file: `config/diagnostics.go`**
- Remove Sprites-specific validation

**Modified file: `loop/defaults.go`**
- Remove the entire `if runtimeEnabled("sprites")` block
- Plugin discovery (phase 4) handles registration

**Modified file: `internal/rtcap/registry.go`**
- Remove `SpritesCaps` variable
- Remove Sprites from `NewRegistry()` pre-registration
- Capabilities now come from plugin `initialize` response

**Modified file: `go.mod`**
- Remove `github.com/superfly/sprites-go` dependency
- Run `go mod tidy`

**Modified files: tests**
- Update `dispatcher/factory_test.go` — remove Sprites routing tests (factory may be removed entirely if unused)
- Update `config/config_test.go` — remove Sprites config tests
- Update `loop/loop_test.go`, `loop/schedule_test.go` — replace Sprites references with generic plugin test fixtures
- Update any other tests referencing Sprites directly

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.4` | Pure deletion — mechanical, no judgment needed |

## Verification

### Static
- `go build ./...` — no Sprites imports remain
- `go vet ./...`
- `go mod tidy` — sprites-go no longer in go.sum
- `grep -r "sprites" --include="*.go"` — only test fixtures and plugin discovery references

### Runtime
- `pnpm build && pnpm check` — full suite passes
- `go test ./...` — all tests pass without Sprites built in
- Process runtime still works as before (no regression)

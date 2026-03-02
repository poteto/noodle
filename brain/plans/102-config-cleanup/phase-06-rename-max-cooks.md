---
todo: 106
---

# Phase 6 — Rename `max_cooks` to `max_concurrency`

Back to [[plans/102-config-cleanup/overview]]

## Goal

Rename the concurrency limit from the internal term "max_cooks" to the user-facing term "max_concurrency" everywhere: TOML key, Go struct field, mise.json field, docs, and examples.

## Changes

### `config/types_defaults.go`
- Rename `MaxCooks int` to `MaxConcurrency int` in `ConcurrencyConfig`
- Change TOML tag from `toml:"max_cooks"` to `toml:"max_concurrency"`
- Update `DefaultConfig()` field name

### `config/parse.go`
- Update default-setting logic to reference `MaxConcurrency`
- Update validation to reference `MaxConcurrency`

### `config/config_test.go`
- Update all `MaxCooks` references to `MaxConcurrency`
- Update TOML test strings from `max_cooks` to `max_concurrency`

### `loop/cook_spawn.go`
- Update `atMaxConcurrency()` (or whatever uses `cfg.Concurrency.MaxCooks`) to `cfg.Concurrency.MaxConcurrency`

### `mise/builder.go`
- Update reference to `config.Concurrency.MaxCooks`

### `mise/types.go`
- Rename `MaxCooks` field in resource snapshot to `MaxConcurrency`
- Update JSON tag if present

### `internal/schemadoc/specs.go`
- Rename `resources.max_cooks` to `resources.max_concurrency`
- Update `max_cooks` in status.json docs

### `generate/skill_noodle.go`
- Update field description key from `concurrency.max_cooks` to `concurrency.max_concurrency`

### `loop/state_snapshot.go`
- Update MaxCooks reference in snapshot building

### `loop/control.go` + `loop/control_orders.go`
- Rename any `max_cooks` references in control action routing and error messages

### `internal/statusfile/statusfile.go`
- Rename `max_cooks` in status JSON output

### `internal/snapshot/types.go`
- Rename `MaxCooks` in API snapshot type

### `ui/src/client/generated-types.ts`
- Rename `max_cooks` / `maxCooks` in generated UI types (may auto-regenerate from Go types)

### `loop/reconcile.go`
- Rename any `MaxCooks` references in concurrency control

### `docs/reference/configuration.md`
- Rename `max_cooks` to `max_concurrency` in docs

### `docs/concepts/scheduling.md`
- Update `max_cooks` references

### `docs/concepts/runtimes.md`
- Update `max_cooks` references

### `examples/multi-skill/.noodle.toml`
- Rename `max_cooks = 2` to `max_concurrency = 2`

### `.noodle.toml` (root project config)
- Rename if present

### `.agents/skills/noodle/references/config-schema.md`
- Update field name

### Test fixtures
- Update any `max_cooks` in `loop/testdata/*/.noodle.toml`

## Data Structures

`ConcurrencyConfig.MaxCooks` becomes `ConcurrencyConfig.MaxConcurrency`. Resource snapshot's `MaxCooks` becomes `MaxConcurrency`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical rename, many files but no judgment needed |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `max_cooks` or `MaxCooks` remain anywhere (grep the entire repo, including UI types)

### Runtime
- `go test ./...` — all tests pass with renamed field
- Parse `.noodle.toml` with `max_concurrency = 4` — works

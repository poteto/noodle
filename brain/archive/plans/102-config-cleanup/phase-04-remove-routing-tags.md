---
todo: 102
---

# Phase 4 — Remove `routing.tags` From Config

Back to [[plans/102-config-cleanup/overview]]

## Goal

Remove tag-based routing overrides from the config. The scheduler skill decides routing per stage — tag-based config routing is unnecessary indirection. `routing.defaults` stays.

## Changes

### `config/types_defaults.go`
- Remove `Tags map[string]ModelPolicy` from `RoutingConfig`
- Remove tags default from `DefaultConfig()`

### `config/parse.go`
- Remove `routing.tags` initialization/default-setting logic

### `config/config_test.go`
- Remove tests asserting routing.tags parsing and defaults

### `mise/builder.go`
- Remove tag iteration from routing snapshot builder (lines ~88-95)
- `RoutingSnapshot` should only carry defaults, no tags map

### `mise/types.go`
- Remove `Tags map[string]RoutingPolicy` from `RoutingSnapshot`

### `internal/orderx/orders.go`
- Remove tag-based routing fallback (line ~93 where it looks up tag policy)
- Stage routing falls back to `routing.defaults` only

### `internal/schemadoc/specs.go`
- Remove `routing.tags{}.provider` and `routing.tags{}.model` entries

### `generate/skill_noodle.go`
- Remove `routing.tags` field description

### `docs/reference/configuration.md`
- Remove routing tags documentation section

### `docs/cookbook/model-routing.md`
- Remove tag-based routing examples; keep defaults-only examples

### `docs/cookbook/multi-stage-pipeline.md`
- Remove `routing.tags.review` example

### `docs/concepts/scheduling.md`
- Remove routing.tags references

### `examples/multi-skill/README.md`
- Remove `routing.tags.*` examples and narrative

### `examples/multi-skill/.noodle.toml`
- Remove `[routing.tags.review]` section

### `.agents/skills/noodle/SKILL.md`
- Remove `routing.tags` entry from the config field table

### `.agents/skills/noodle/references/config-schema.md`
- Remove routing.tags documentation

### Test fixtures
- Remove routing.tags from any `loop/testdata/*/.noodle.toml` files

## Data Structures

`RoutingConfig` shrinks to just `Defaults ModelPolicy`. `RoutingSnapshot` loses its `Tags` map.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Touches many files across packages; needs judgment for doc updates |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `routing.tags`, `RoutingConfig.Tags`, or tag-based routing remain (grep for `\.Tags`, `routing.tags`)

### Runtime
- `go test ./config/...` — config parsing works without tags
- `go test ./mise/...` — mise builder generates routing snapshot without tags
- `go test ./internal/orderx/...` — order routing uses defaults only
- `go test ./loop/...` — integration tests pass

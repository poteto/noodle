Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 6: Unify Routing Types

## Goal

Deduplicate the structurally identical `config.ModelPolicy` and `mise.RoutingPolicy` types while preserving boundary discipline. These types live at different system boundaries: `ModelPolicy` is config-bound (TOML deserialization), `RoutingPolicy` is part of the emitted brief schema consumed by schedulers (JSON). Merging them directly would couple config evolution to the mise.json API.

## Changes

- **Keep both external types.** `config.ModelPolicy` stays as the config DTO. `mise.RoutingPolicy` stays as the brief DTO.
- **`mise/builder.go`** — replace the manual field-by-field copy (`RoutingPolicy{Provider: policy.Provider, Model: policy.Model}`) with a shared conversion helper to reduce drift risk.
- **Consider a shared internal value type** in `internal/` if the field set grows beyond 2 fields in the future. For now, the types are small enough that explicit conversion is clearer than a shared type with dual serialization tags.

## Data Structures

No types deleted. Conversion made explicit rather than implicit.

## Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`
- Small, focused change.

## Verification

### Static
- `go build ./config/... ./mise/...` — compiles
- `go test ./config/... ./mise/...` — tests pass
- `go vet ./...` — clean

### Runtime
- `go test ./...` — full suite passes

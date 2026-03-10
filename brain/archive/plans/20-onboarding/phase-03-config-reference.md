Back to [[plans/20-onboarding/overview]]

# Phase 3 — Configuration Reference

## Goal

Write the `.noodle.toml` configuration reference page. This ships early because concept docs (phases 4-6) and the getting-started tutorial (phase 8) all link to it.

## Changes

- **`docs/reference/configuration.md`** — Full `.noodle.toml` field reference:
  - Every section and field, generated from `config/types_defaults.go`
  - Default values
  - Common configurations (routing, concurrency, runtimes, skill paths)
  - Adapters overview
  - Minimal vs full config examples

## Data structures

- `config.Config` — document the struct tree from `config/types_defaults.go`

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.4` | Mechanical extraction from Go struct definitions |

## Verification

### Static
- Every field in `config.Config` (from `config/types_defaults.go`) is documented
- Default values match `DefaultConfig()` output
- Page builds in VitePress without errors

### Runtime
- Cross-reference documented fields against a generated `.noodle.toml` from first run

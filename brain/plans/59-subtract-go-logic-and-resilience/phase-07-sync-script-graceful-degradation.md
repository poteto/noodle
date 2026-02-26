Back to [[plans/59-subtract-go-logic-and-resilience/overview]]

# Phase 7: Sync script graceful degradation

Covers: #62

## Goal

When the backlog adapter sync script is misconfigured or missing, degrade to an empty backlog with a warning instead of crashing the cycle.

## Changes

- `loop/loop.go` (~lines 378-382) — Replace the `return Queue{}, false, fmt.Errorf(...)` with: log a warning about the missing sync script, emit a feed event, and continue with an empty queue. The cycle proceeds — if the schedule skill has other work (non-backlog items, maintenance tasks), it can still schedule them.
- `loop/util.go` (~lines 196-210) — Delete or simplify `shouldRecoverMissingSyncScripts()`. With graceful degradation, the "recovery" concept is no longer needed — the loop just treats missing sync as empty backlog.

## Data structures

No changes.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Small behavioral change — delete function, change error to warning |

## Verification

### Static
- `go test ./loop/...` passes
- `go vet ./...` clean
- New test: mise build with missing sync script → returns brief with empty backlog + warning (not error)
- New test: cycle continues after missing sync script (not halted)
- Grep for `shouldRecoverMissingSyncScripts` — zero hits (or simplified)

### Runtime
- Configure `.noodle.toml` with a nonexistent sync script path, run `noodle start --once` → cycle runs, warning logged, no crash

# Phase 6: Unified Mode and Control API

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Replace split oversight controls with one global mode that deterministically gates schedule, dispatch, retry, and merge behavior.

## Changes

- Add config `mode = auto | supervised | manual`
- Remove `autonomy` and vestigial `schedule.run` from backend contract
- Delete control action `autonomy`; `mode` is the only oversight control
- Add manual `dispatch` control action for explicit launch in manual mode
- Add mode transition contract in status/snapshot (`requested_mode`, `effective_mode`, `requested_by`, `reason`, `applied_at`)
- Add blocked-action reason codes for schedule/dispatch/retry/merge gates

### Alternatives considered

1. Keep two controls and wire missing behavior.
2. Single global mode with explicit gate semantics.
3. Two modes only + per-skill overrides.

Chosen: **(2)** for operator clarity and deterministic gate evaluation.

### In-flight semantics

- Add `mode_epoch` to canonical state.
- Effects are stamped with creation `mode_epoch`.
- Executor revalidates epoch before commit; mismatches emit deterministic cancellation/defer events.
- Implementation note: comparison representation for `mode_epoch` (monotonic integer vs. comparable token) is decided during implementation, but the invariant is fixed: epochs must be strictly ordered and unambiguous across restarts.

## Data Structures

- `RunMode`
- `ModeGate`
- `ModeTransitionRecord`
- `ControlAction` enum updates
- `ModeEpoch`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Architecture / judgment | `claude` | `claude-opus-4-6` | Decide `mode_epoch` comparison semantics and cancellation invariants |
| Implementation | `codex` | `gpt-5.4` | Config/control wiring based on finalized semantics |

## Verification

### Static

- Parse/validation rejects unsupported mode values
- No reads of removed `autonomy`/`schedule.run` paths in loop logic
- `go test ./... && go vet ./...`

### Runtime

- End-of-phase e2e smoke test: `pnpm test:smoke`
- Mode matrix integration tests:
  - auto: full automation with per-skill merge permissions
  - supervised: auto schedule/dispatch with mandatory review parking
  - manual: no auto schedule/dispatch/retry, explicit dispatch required
- In-flight mode change tests with queued and running effects
- UX acceptance: blocked actions include `why` and exact next control command

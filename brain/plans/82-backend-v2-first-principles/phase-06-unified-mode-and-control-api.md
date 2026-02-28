# Phase 6: Unified Mode and Control API

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Replace split oversight controls with one global mode that deterministically gates schedule, dispatch, retry, and merge behavior.

## Changes

- Add config `mode = auto | supervised | manual`
- Remove `autonomy` and vestigial `schedule.run` from backend contract
- Replace control action `autonomy` with `mode`
- Add manual `dispatch` control action for explicit launch in manual mode
- Update status/snapshot fields to surface `mode` only

## Data Structures

- `RunMode`
- `ModeGate` helper for schedule/dispatch/retry/merge checks
- `ControlAction` enum updates

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Config/control plumbing and gate wiring |

## Verification

### Static

- Parse/validation rejects unsupported mode values
- No reads of removed `autonomy`/`schedule.run` paths in loop logic
- `go test ./... && go vet ./...`

### Runtime

- Mode matrix integration tests:
  - auto: full automation with per-skill merge permissions
  - supervised: auto schedule/dispatch with mandatory review parking
  - manual: no auto schedule/dispatch/retry, explicit dispatch required
- Edge cases: runtime mode change during active sessions

Back to [[plans/88-subagent-tracking-v2/overview]]

# Phase 4: Steer-Agent Control Hardening

## Goal

Harden agent steering behavior while keeping a single control path (`/api/control` -> loop command processing).

## Changes

- Tighten `steer-agent` validation rules (`agent exists`, `steerable`, `running`, `parent session active`).
- Add provider-specific timeout/retry policy inside loop-side steering execution, not server handler shortcuts.
- Standardize failure responses so UI gets actionable, typed failure reasons.

## Data Structures

- `ControlCommand` extension: `{Action: "steer-agent", Target, AgentID, Prompt}`
- `SteerFailure`: mapped onto existing control ack failure contract (`code`, `message`, `retryable`)

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Error semantics and control-path design judgment.

## Verification

### Static
- `go test ./server/... ./loop/...`
- `go vet ./...`

### Runtime
- Attempt steer on non-steerable and completed agents; verify deterministic failures.
- Steer active Codex child; verify message lands on target and appears in feed.

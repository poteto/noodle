Back to [[plans/88-subagent-tracking-v2/overview]]

# Phase 2: Agent Identity Reconciliation Hardening

## Goal

Guarantee canonical `AgentID` handling end-to-end so one logical agent always materializes as one node/event stream.

## Changes

- Build on plan 84 canonical-ID rules; do not reintroduce provisional-ID persistence.
- Harden only post-v1 gaps: cross-source reconciliation (in-band + out-of-band), legacy replay compatibility, and deterministic metadata precedence under mixed ordering.
- Update snapshot materialization logic to upsert by canonical `AgentID` and merge metadata deterministically across source modes.

Relationship to Plan 84:
- This phase does not replace v1 parser reconciliation mechanics.
- This phase adds cross-source/upgrade compatibility rules on top of v1 so plan-84-era logs and v2 mixed-source logs resolve to the same canonical node identity model.

## Data Structures

- `AgentIdentity`: `{CanonicalID, ProvisionalIDs, ParentCanonicalID}`
- `AgentNode` upsert policy keyed by canonical ID

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Architecture/judgment-heavy identity and reconciliation rules.

## Verification

### Static
- `go test ./parse/... ./internal/snapshot/...`
- `go vet ./...`

### Runtime
- Replay mixed-order fixtures where spawn/progress/complete arrive in different orders; verify no duplicate nodes.
- Verify parent-child links remain stable when provisional IDs are present in source logs.
- Replay plan-84-era logs under v2 reconciliation rules and verify canonical IDs/nodes remain stable (no reinterpretation to provisional IDs).

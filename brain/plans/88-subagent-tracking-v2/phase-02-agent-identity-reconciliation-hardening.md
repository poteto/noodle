Back to [[plans/88-subagent-tracking-v2/overview]]

# Phase 2: Agent Identity Reconciliation Hardening

## Goal

Guarantee canonical `AgentID` handling end-to-end so one logical agent always materializes as one node/event stream.

## Changes

- Enforce parser-level rule: provisional correlation IDs never persist as canonical `AgentID`.
- Normalize Codex and Claude spawn/progress/complete reconciliation so duplicate spawn-like signals merge on canonical identity.
- Update snapshot materialization logic to upsert by canonical `AgentID` and merge metadata deterministically.

Relationship to Plan 84:
- Replace the v1 provisional map behavior with this canonical reconciliation contract (single source of truth), rather than maintaining two parallel identity mechanisms.

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

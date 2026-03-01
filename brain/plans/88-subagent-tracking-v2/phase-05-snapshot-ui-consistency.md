Back to [[plans/88-subagent-tracking-v2/overview]]

# Phase 5: Snapshot and UI Consistency

## Goal

Keep tree/feed/agent-chat state coherent when events arrive late, out-of-order, or from mixed in-band and out-of-band sources.

## Changes

- Update snapshot builder ordering/upsert logic to produce stable agent status and current action under event reordering.
- Update UI event filtering/state handling to avoid duplicate rows and stale optimistic entries for steering.
- Ensure steer-input enabled/disabled state derives from reconciled agent status, not transient single-event assumptions.

Determinism rules:
- Ordering key: `(event_timestamp, ingest_sequence)` where ingest_sequence is monotonic per source
- Terminal precedence: `completed`/`errored` are terminal and absorb late non-terminal progress updates
- Equal timestamp tie-breaker uses ingest_sequence, not map iteration order

## Data Structures

- `AgentStatusProjection`: derived `{Status, CurrentAction, LastEventAt}`
- UI state keying by canonical `{session_id, agent_id}`

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Interaction and state consistency judgment.

## Verification

### Static
- `go test ./internal/snapshot/...`
- `cd ui && pnpm tsc --noEmit && pnpm test`

### Runtime
- Replay reordered event fixtures; verify stable tree/feed output.
- Manual UI pass: switch parent/agent views during live updates and verify no flicker/duplication.

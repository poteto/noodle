Back to [[plans/88-subagent-tracking-v2/overview]]

# Phase 5: Snapshot and UI Consistency

## Goal

Keep tree/feed/agent-chat state coherent when events arrive late, out-of-order, or from mixed in-band and out-of-band sources.

## Changes

- Update snapshot builder ordering/upsert logic to produce stable agent status and current action under event reordering.
- Update UI event filtering/state handling to avoid duplicate rows and stale optimistic entries for steering.
- Ensure steer-input enabled/disabled state derives from reconciled agent status, not transient single-event assumptions.

Determinism rules:
- Dedupe key: `(source_key, source_offset)` for out-of-band derived events; duplicates are discarded before projection
- Ordering key: `(event_timestamp, source_key, ingest_sequence)` where ingest_sequence is monotonic per source
- Terminal precedence: `completed`/`errored` are terminal and absorb late non-terminal progress updates
- Equal timestamp tie-breaker uses `(source_key, ingest_sequence)`, not map iteration order

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
- Replay duplicate out-of-band fixture lines; verify tree/feed rows remain single-instance by dedupe key.
- Manual UI pass: switch parent/agent views during live updates and verify no flicker/duplication.

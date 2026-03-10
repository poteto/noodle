Back to [[plans/88-subagent-tracking-v2/overview]]

# Phase 1: Out-of-Band Agent Ingestion

## Goal

Ingest provider-owned out-of-band files (Codex child sessions and Claude team inboxes) through the existing canonical parser and event pipeline.

## Changes

- Add a session-scoped ingestion coordinator in backend runtime/dispatcher flow to discover and ingest:
  - Codex child session JSONL files referenced by known child thread IDs
  - Claude team inbox files for known team members
- Feed discovered lines through existing parse + canonical event plumbing instead of building a parallel channel.
- Persist per-source read offsets so restarts continue from last processed position.
- Add emit-time dedupe contract for replay safety: every out-of-band line carries a deterministic ingest identity (`source_key`, `source_offset`) and duplicate identities are dropped before appending session events.

Codex child path resolution rules:
- Primary lookup by thread ID filename under date-partitioned tree (`~/.codex/sessions/YYYY/MM/DD/{thread_id}.jsonl`)
- Search window: today, yesterday, tomorrow partitions (handles midnight rollover)
- If unresolved, retry with bounded backoff until timeout; emit observable warning on timeout
- Retry budget: exponential backoff from 250ms to 5s, max 90s total per unresolved child before timeout warning
- Discovery order is deterministic: exact partition match -> adjacent partitions -> bounded glob fallback inside the 3-day window

Checkpoint persistence rules:
- Store checkpoints in runtime dir (for example: `.noodle/sessions/{parent}/ingestion-checkpoints.ndjson`)
- Write checkpoint updates atomically (temp file + rename)
- Include source fingerprint (`provider`, `path`, `agent_id`) to prevent misapplying offsets after source remap
- Recovery rule: on corrupt checkpoint read, rename checkpoint to `.corrupt.<ts>`, emit warning event, rebuild per-source high-water marks from existing session event log (`source_key` + `source_offset`), then replay from safe boundary (offset 0) with dedupe guard active

## Data Structures

- `AgentSourceRef`: `{Provider, ParentSessionID, AgentID, SourceType, Path}`
- `IngestionCheckpoint`: `{SourceKey, Provider, Path, AgentID, Offset, HighWaterOffset, UpdatedAt}`
- `IngestIdentity`: `{SourceKey, SourceOffset}` attached to out-of-band derived events for dedupe/replay safety

## Routing

- Provider: `codex`
- Model: `gpt-5.4`
- Mechanical ingestion scaffolding with clear boundary contracts.

## Verification

### Static
- `go test ./dispatcher/... ./server/... ./parse/...`
- `go vet ./...`

### Runtime
- Run session fixtures where child activity only exists in out-of-band files; verify events appear in feed/tree.
- Restart during ingestion; verify no duplicate replay and no missed tail lines.
- Midnight rollover test: spawn near day boundary and resolve child file across partition directories.
- Corrupt/partial checkpoint recovery test: restart does not skip unread lines.
- Corrupt checkpoint replay test: replay-from-zero produces zero duplicate emitted events because dedupe identity is enforced.

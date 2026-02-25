Back to [[archived_plans/38-resilient-skill-resolution/overview]]

# Phase 5: Registry rebuild and queue audit

**Routing:** `claude` / `claude-opus-4-6` — queue state mutation, event generation, correctness matters

## Goal

When the skill watcher triggers a registry rebuild, the current queue may contain items referencing skills that no longer exist. Audit the queue: drop stale items and emit events so the user knows what happened and why.

Also fix the registry error path: currently `runCycleMaintenance()` hard-returns `registryErr` which exits the loop. It should route through runtime repair instead so the loop can self-heal.

## Changes

**Enrich `rebuildRegistry()` (defined in phase 4):**
- `loop/loop.go` — after rebuilding, call `auditQueue()` (new method). Return a `RegistryDiff` so callers can see what changed.

**Fix registry error path in cycle:**
- `loop/loop.go` — change `runCycleMaintenance()` (line 192): instead of `return false, l.registryErr`, log the error to stderr, skip the current cycle (`return false, nil`), and set a `registryStale` flag so the next cycle retries discovery. If discovery fails 3 consecutive times, return the error as fatal. This prevents a single transient filesystem error from killing the loop while still surfacing persistent problems.

**Queue audit after rebuild:**
- `loop/queue_audit.go` (new file) — `auditQueue()` reads `queue.json`, validates each item against the new registry. Items with unknown task types are removed. Returns list of dropped items with reasons.
- Write audit events to a `queue-events.ndjson` file in `.noodle/` so the TUI can surface them (phase 8).
- Retention: truncate `queue-events.ndjson` to last 200 lines on each write to prevent unbounded growth.

**Audit event format:**
```json
{"at":"2026-02-24T15:04:05Z","type":"queue_drop","target":"todo-3","skill":"deploy","reason":"skill no longer exists"}
{"at":"2026-02-24T15:04:05Z","type":"registry_rebuild","added":["deploy"],"removed":["staging"]}
```

**Stderr logging:**
- Log each dropped item to stderr: `"dropped queue item %q: skill %q no longer exists"`
- Log registry changes: `"skill registry rebuilt: added [X], removed [Y]"`

## Quality reference inventory for this phase

These `quality` references are runtime/data-pipeline touches that should be
classified during queue audit work as either intentional (verdict domain) or
obsolete (skill-name coupling):

- Runtime verdict flow: `loop/quality.go`, `mise/quality.go`, `mise/builder.go`
- Runtime verdict storage contract: `dispatcher/preamble.go` (`.noodle/quality/`)
- Autonomy/review behavior docs in code: `config/config.go`
- Generated product model text: `generate/skill_noodle.go`
- Regression coverage for verdict handling: `loop/loop_test.go`, `mise/quality_test.go`

Phase acceptance for this inventory:
- Keep verdict-domain references that describe review outputs and history.
- Remove or redirect references that still imply a standalone `quality` skill.

## Data structures

- `QueueAuditEvent` — struct for NDJSON events (type, target, skill, reason, timestamp)
- `RegistryDiff` — tracks added/removed skills between old and new registry

## Verification

```bash
go test ./loop/... && go vet ./...
```

Unit tests:
- Start with registry containing skill X. Write a queue item targeting X. Delete skill X from disk. Trigger rebuild. Verify item is removed from queue.json. Verify event written to queue-events.ndjson. Verify stderr output.
- Registry discovery failure during rebuild: verify error is logged, cycle skipped, and retried next cycle. After 3 consecutive failures, verify fatal error propagated.
- queue-events.ndjson with 300 lines: after write, verify truncated to 200.

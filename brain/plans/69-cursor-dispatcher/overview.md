---
id: 69
created: 2026-02-27
status: ready
---

# Cursor Dispatcher

## Context

Noodle dispatches agent sessions via local processes (ProcessDispatcher) and remote Sprites VMs (SpritesDispatcher). Both use streaming backends — real-time stdout parsed into events. Cursor Cloud Agent exposes a REST polling API instead: launch an agent, poll for status, fetch results. Plan 27 laid the groundwork with `PollingBackend` interface and a `CursorBackend` stub, but deferred the actual implementation (phase 4) until the backend became real.

Cursor's Cloud Agent API (v0, beta) supports: launching agents against GitHub repos, polling status (CREATING → RUNNING → FINISHED/ERROR/EXPIRED), fetching conversation history, and webhook notifications on status changes.

## Scope

**In:**
- Real `CursorBackend` HTTP client replacing the stub (all `PollingBackend` methods)
- `PollingDispatcher` + `pollingSession` — generic dispatcher for any `PollingBackend`
- `CursorRuntime` + factory wiring in `defaults.go`
- Webhook receiver endpoint on Noodle's HTTP server for status change notifications
- Config: `AvailableRuntimes()` includes cursor when API key is set

**Out:**
- PR creation/management (branch-based sync only, as with Sprites)
- First-run auth token setup UX (assume `CURSOR_API_KEY` is configured)
- Agent steering/follow-up (Cursor sessions are not steerable — uses `NoopController`)
- Long-running agent toggle (not exposed in Cursor's API yet)

## Constraints

- Cursor API uses Basic auth (API key as username, empty password)
- Agent pushes to an auto-generated branch (target.branchName in the response) — no PR by default
- `PollLaunchConfig` already has `Prompt`, `Repository`, `Model`, `Branch` — maps directly to Cursor's POST body
- `Launch` must return structured metadata (`LaunchResult{RemoteID, TargetBranch}`) — branch is immutable launch-time data, not just poll-time data. Persist `remote_id` and `target_branch` in `spawn.json` immediately at launch for crash recovery.
- `writeSyncResult` uses non-atomic read-modify-write on `spawn.json`. Use `filex.WriteFileAtomic` for atomic rewrite of `spawn.json` (not a separate `sync.json`) — loop merge readers already consume `sync` from `spawn.json` (`cook_merge.go:110`), so a separate file would require updating all consumers. Atomic rewrite avoids crash-window corruption while keeping a single source of truth. (Note: concurrent-writer TOCTOU is not a concern — the pollingSession poll loop is single-owner.)
- Webhooks require a publicly accessible URL — Noodle runs on localhost by default. Polling is the primary completion mechanism; webhooks are an optimization for users who expose their server.
- HTTP errors from Cursor must be classified as retryable (429, 5xx, transient network) vs terminal (401, 403, 404, 410). Terminal errors fail the session immediately; retryable errors use exponential backoff with jitter and `Retry-After` support.

## Alternatives considered

**Completion detection:**
1. **Polling only** — simple, works universally, no networking requirements. Downside: latency up to poll interval, API calls consumed.
2. **Webhook primary + polling fallback** — lower latency, fewer API calls. Downside: requires public URL (tunnel setup for localhost).
3. **Long-polling / SSE** — Cursor's API doesn't support this.

**Chosen: Polling as primary, webhook as optional optimization (phase 7).** Webhooks can layer on without architectural changes. The pollingSession isolates detection from the rest of the system.

**Branch sync:**
1. **Pull target branch at completion** — write SyncResult with Cursor's target branch. Loop's existing merge flow handles it via `MergeRemoteBranch`.
2. **Auto-create PR and merge** — unnecessary complexity; branch-based merge aligns with Sprites flow.

**Chosen: Branch-based sync using existing `MergeRemoteBranch`.**

**PollStatus return type:**
1. **Enrich to `PollResult` struct** — carries status, branch, summary. Generic enough for future backends. Only 2 existing implementors (stub + test), trivial migration.
2. **Separate metadata method** — adds interface surface for no benefit.
3. **Split Launch to return `LaunchResult` + keep PollResult for mutable state** — captures immutable metadata (target branch) at launch time, not just poll time. More robust: branch is available even if final poll fails.

**Chosen: Both `LaunchResult` from `Launch` and `PollResult` from `PollStatus`.** Branch is captured at launch (immutable) and confirmed at poll (mutable). `LaunchResult` persisted immediately for crash recovery.

## Applicable skills

- `go-best-practices` — concurrency (polling goroutine), HTTP client patterns, lifecycle shutdown
- `testing` — httptest for HTTP client, race detector for concurrency

## Phases

- [[plans/69-cursor-dispatcher/phase-01-enrich-pollingbackend-with-pollresult]]
- [[plans/69-cursor-dispatcher/phase-02-cursor-http-client]]
- [[plans/69-cursor-dispatcher/phase-03-cursorbackend-implementation]]
- [[plans/69-cursor-dispatcher/phase-04-pollingsession]]
- [[plans/69-cursor-dispatcher/phase-05-pollingdispatcher]]
- [[plans/69-cursor-dispatcher/phase-06-cursorruntime-and-factory-wiring]]
- [[plans/69-cursor-dispatcher/phase-07-webhook-receiver-endpoint]]

## Review revisions (2026-02-27)

6 independent reviewers (Codex, 2 rounds) all returned **Revise**. Consensus issues addressed:

1. **Crash recovery** — persist `remote_id` in spawn.json at launch; add polling-runtime recovery that re-attaches to live Cursor agents on restart. Added to phases 1, 5, 6.
2. **Atomic spawn.json writes** — replace `writeSyncResult` read-modify-write with atomic temp+rename or separate `sync.json`. Added to phase 4 constraints.
3. **Error classification** — retryable (429, 5xx) vs terminal (401, 403, 404). Terminal fails fast; retryable uses backoff. Added to phases 2, 4.
4. **Post-launch rollback** — if local steps fail after remote launch, call Stop/Delete best-effort. Added to phase 5.
5. **Kill() terminal race** — single-owner state machine in poll loop. Kill() signals stop; poll loop performs final transition. Added to phase 4.
6. **Webhook notifier wiring** — session registry (mutex-protected, register/unregister lifecycle) scaffolded in phases 4-5, wired through server options in phase 6. Added to phases 4, 5, 6, 7.
7. **Launch returns LaunchResult** — `Launch` returns `LaunchResult{RemoteID, TargetBranch}` instead of bare string. Target branch persisted immediately. Added to phase 1.
8. **Monitor/heartbeat integration** — pollingSession writes heartbeat on each poll and event-writer records. Added to phase 4.
9. **Config validation** — validate repository non-empty, `webhook_secret_env` follows env-key pattern. Added to phases 6, 7.
10. **Terminal cleanup** — call Delete on completed/failed/expired. Added to phase 4.

## Review revisions round 3 (2026-02-27)

3 independent reviewers (Codex, round 3) all returned **Revise**. Consensus issues addressed:

1. **Canonical terminal events for monitor compatibility** — pollingSession must write `canonical.ndjson` terminal events (EventResult with completion/failure) so monitor claims can derive correct session status. Without this, recovered successful sessions would be misclassified as failed. Added to phase 4.
2. **Recovery incompatible with PID-based adopted-session pruning** — `refreshAdoptedTargets` uses `SessionPIDAlive` which always fails for remote sessions. Added heartbeat-based liveness check for non-process runtimes to phase 6.
3. **OrderID not populated during recovery** — `RecoveredSession.OrderID` must be set from `spawn.json` for reconcile to map sessions to orders. Added `order_id` persistence at dispatch time and recovery population to phases 5/6.
4. **APIError missing RetryAfter field** — pollingSession needs server-specified backoff duration from 429 responses. Added `RetryAfter time.Duration` to `APIError` in phase 1, wired through phase 2 HTTP client, consumed in phase 4 poll loop.
5. **Webhook notifier wiring through cmd_start.go** — concrete boundary path specified: `defaultDependencies()` → `runWebServer()` parameter → `server.Options.SessionNotifier`. Added to phase 6.
6. **Context cancellation leaks remote agents** — cancel path must call `backend.Stop`/`Delete` before exiting, not just mark cancelled. Added to phase 4.
7. **Sync persistence locked to atomic spawn.json** — eliminated sync.json alternative; `filex.WriteFileAtomic` on `spawn.json` is the single path. Updated overview constraint.
8. **Registry collision semantics** — `Register` with existing key returns error (rejects duplicate). Added to phase 4.
9. **410 (Gone) in terminal errors** — added to phase 1 and phase 4 terminal error list.
10. **Recovery transient error handling** — 429/5xx during startup recovery treated as "still alive" (adopt for polling), not "failed". Added to phase 6.

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Manual: configure `.noodle.toml` with `[runtime.cursor]` block (api_key_env, repository), set `runtime.default = "cursor"`, dispatch a session, verify polling detects completion and branch is merged.

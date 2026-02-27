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
- `writeSyncResult` uses non-atomic read-modify-write on `spawn.json`. Replace with atomic writes (`filex.WriteFileAtomic`) or a separate `sync.json` file to avoid TOCTOU races with concurrent loop/monitor readers.
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

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Manual: configure `.noodle.toml` with `[runtime.cursor]` block (api_key_env, repository), set `runtime.default = "cursor"`, dispatch a session, verify polling detects completion and branch is merged.

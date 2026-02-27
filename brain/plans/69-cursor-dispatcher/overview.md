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
- `writeSyncResult` in `dispatcher/sprites_dispatcher.go` is package-level and reusable by `pollingSession`
- `writeDispatchMetadata` creates `spawn.json`; `writeSyncResult` patches it with a `sync` field; loop reads `sync.Branch` and calls `Worktree.MergeRemoteBranch`
- Webhooks require a publicly accessible URL — Noodle runs on localhost by default. Polling is the primary completion mechanism; webhooks are an optimization for users who expose their server.

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

**Chosen: `PollResult` struct.** Per redesign-from-first-principles, this is what we'd build knowing Cursor exists.

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

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Manual: configure `.noodle.toml` with `[runtime.cursor]` block (api_key_env, repository), set `runtime.default = "cursor"`, dispatch a session, verify polling detects completion and branch is merged.

Back to [[plans/90-interactive-sessions/overview]]

# Phase 5 — Crash Recovery

## Goal

Interactive sessions survive loop restarts. If the loop crashes and restarts while an interactive session's process is alive, the session is re-adopted into `l.chats` and becomes stoppable from the UI. Recovered sessions are **stop-only, not steerable** — the stdin pipe is lost on restart, so `SendMessage()` cannot work.

## Changes

**`dispatcher/dispatch_metadata.go`** (or wherever `SessionMeta` is defined) — Add `Kind SessionKind` and `Name string` fields to the existing `SessionMeta` struct. The monitor already reads/writes `SessionMeta` to `meta.json` — adding fields here ensures they survive monitor rewrites (unlike a sidecar file, which the monitor would overwrite or ignore).

**`loop/chat.go`** — When spawning an interactive session, set `Kind` and `Name` on the session's metadata before dispatch. These flow into `meta.json` via the existing monitor write path.

**`runtime/process_recover.go`** — Extend recovery to read `Kind` from `SessionMeta`. When `Kind == "interactive"`, mark the recovered session for interactive adoption (instead of requiring `OrderID`). The recovered handle has no live controller — steer operations will fail, but stop/kill work via process signals.

**`loop/reconcile.go`** — Extend reconciliation to adopt recovered interactive sessions into `l.chats`. Currently `reconcile` only indexes sessions when `OrderID != ""`. Add a branch: if recovered session has `Kind == KindInteractive`, create a `chatHandle` and register it. Mark it as recovered so the snapshot can surface a "recovered" status to the UI.

## Data Structures

- `SessionMeta.Kind SessionKind` — added to existing struct
- `SessionMeta.Name string` — added to existing struct
- `chatHandle.recovered bool` — flag for UI status

## Routing

- **Provider:** `codex`
- **Model:** `gpt-5.4`

## Verification

### Static
- `go build ./...`
- `go vet ./...`

### Runtime
- Test: spawn interactive session, verify `meta.json` contains `kind: "interactive"` and `name`
- Test: monitor rewrites `meta.json`, verify `kind` and `name` fields are preserved
- Test: simulate loop restart (kill + restart loop), verify session re-adopted into `l.chats`
- Test: after recovery, verify interactive session appears in snapshot with recovered status
- Test: after recovery, verify `stop` action works (process killed)
- Test: after recovery, verify `steer` action returns error (not steerable — stdin pipe lost)
- Test: order session recovery still works unchanged

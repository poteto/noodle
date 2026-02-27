Back to [[archived_plans/70-scaling-the-loop-redesign/overview]]

# Phase 5: Aggregate mise brief

## Goal

Replace the per-session `ActiveCooks []ActiveCook` in the brief with an `ActiveSummary` aggregate. The scheduler doesn't need to know about individual sessions — it needs to know how many agents are running, by task type and by status. This makes brief generation O(1) for the active section instead of O(sessions).

## Changes

**`mise/types.go`** — Replace `ActiveCooks []ActiveCook` with `ActiveSummary`. Remove `ActiveCook` struct. Update `Brief` struct.

**`mise/builder.go`** — `readSessionState()` is replaced. The builder receives `ActiveSummary` and `RecentHistory` as arguments (passed by the loop when building the brief) instead of scanning session directories. The `sessionmeta.ReadAll()` call is removed from the brief-building hot path.

**`loop/loop.go`** — The loop maintains a `RecentHistory` ring buffer (capped at 20 entries). Appended on stage completion with session ID, order ID, task key, status, duration, and timestamp. This replaces the `sessionmeta.ReadAll()` scan as the source for history in the mise brief and snapshot. The ring buffer is part of the in-memory loop state exported via `State()` (phase 8).

**`loop/loop.go`** — The loop maintains an `ActiveSummary` counter. Incremented on dispatch, decremented on completion. Broken down by task key and status. On startup, hydrate from `Runtime.Recover()` results — adopted/recovered sessions must be counted in the summary, not just newly dispatched ones. Passed to the mise builder.

**`mise/builder_test.go`** — Update tests that assert on `ActiveCooks` to use `ActiveSummary`.

**`internal/schemadoc/specs.go`** — Update the mise schema documentation: remove `active_cooks[].*` field specs, add `active_summary.*` field specs (`total`, `by_task_key`, `by_status`, `by_runtime`).

**`cmd_status.go`** — Update `readSessionSummary()` to derive active count from `status.json`. Note: `status.json` currently has `Active []string` (order IDs, not session counts — see `internal/statusfile/statusfile.go:13-14`) and `MaxCooks int`. Derive active count as `len(status.Active)` with a documented caveat that this is order count, not session count. Or: add an explicit `ActiveSessionCount int` field to `statusfile.Status` and stamp it in `stampStatus()` (`queue.go:12`). This removes the last `sessionmeta.ReadAll()` call from a hot path.

**`mise/types.go`** — Update `MiseBuilder` interface: `Build()` signature changes to accept `ActiveSummary` and `RecentHistory` arguments instead of discovering them internally.

**`loop/defaults.go`** — Update `defaultDependencies` wiring (`defaults.go:77-87`) to pass the new builder signature.

**`cmd_mise.go`** — Update direct `MiseBuilder.Build()` caller (`cmd_mise.go:30-44`) to provide `ActiveSummary` and `RecentHistory` (or zero values for CLI invocation).

**Schedule skill SKILL.md** — Update to reference `ActiveSummary` fields instead of `ActiveCooks` array. The skill already makes aggregate judgments ("are there too many execute agents?") — now the data matches the query.

## Data structures

- `ActiveSummary` — `Total int`, `ByTaskKey map[string]int`, `ByStatus map[string]int`, `ByRuntime map[string]int`
- `HistoryItem` — `SessionID`, `OrderID`, `TaskKey`, `Status`, `Duration`, `CompletedAt`
- `RecentHistory` — ring buffer of `HistoryItem`, capped at 20

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — contract migration across multiple consumers requires judgment about what each consumer actually needs

## Verification

### Static
- `go test ./...` — all tests pass
- `ActiveCook` type no longer exists
- `sessionmeta.ReadAll()` not called from mise builder or cmd_status
- `MiseBuilder` interface signature updated — all callers compile (loop/defaults.go, cmd_mise.go)
- Schedule skill SKILL.md references `ActiveSummary` (low priority — skill doesn't currently reference `active_cooks` directly)
- Schema docs (`schemadoc/specs.go`) reflect new field structure

### Runtime
- Integration test: dispatch 3 sessions with different task keys, verify ActiveSummary counts are correct
- Verify mise.json contains `active_summary` instead of `active_cooks`
- Run schedule skill against the new brief format, verify it produces valid orders (not just "reasonable" — verify the output parses and contains expected task types)
- `noodle status` CLI command shows correct active count without scanning session dirs
- Prompt fidelity: verify cook prompts still include order rationale and context (ActiveSummary migration must not break prompt construction, which uses per-order data not per-session data)
- RecentHistory: verify completed sessions appear in mise.json `recent_history` with correct fields, capped at 20
- ActiveSummary hydration: after restart with 3 recovered sessions, ActiveSummary.Total == 3

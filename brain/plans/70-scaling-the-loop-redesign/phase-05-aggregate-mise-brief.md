Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 5: Aggregate mise brief

## Goal

Replace the per-session `ActiveCooks []ActiveCook` in the brief with an `ActiveSummary` aggregate. The scheduler doesn't need to know about individual sessions — it needs to know how many agents are running, by task type and by status. This makes brief generation O(1) for the active section instead of O(sessions).

## Changes

**`mise/types.go`** — Replace `ActiveCooks []ActiveCook` with `ActiveSummary`. Remove `ActiveCook` struct. Update `Brief` struct.

**`mise/builder.go`** — `readSessionState()` is replaced. The builder receives the active summary from the loop (via a callback or by reading the in-memory state) instead of scanning session directories. The `sessionmeta.ReadAll()` call is removed from the brief-building hot path. History items can be read lazily or capped.

**`loop/loop.go`** — The loop maintains an `ActiveSummary` counter. Incremented on dispatch, decremented on completion. Broken down by task key and status. Passed to the mise builder.

**`mise/builder_test.go`** — Update tests that assert on `ActiveCooks` to use `ActiveSummary`.

**`internal/schemadoc/specs.go`** — Update the mise schema documentation: remove `active_cooks[].*` field specs, add `active_summary.*` field specs (`total`, `by_task_key`, `by_status`, `by_runtime`).

**`cmd_status.go`** — Update `readSessionSummary()` to derive active count from `status.json` or orders.json (active stage count) instead of scanning session directories. This removes the last `sessionmeta.ReadAll()` call from a hot path.

**Schedule skill SKILL.md** — Update to reference `ActiveSummary` fields instead of `ActiveCooks` array. The skill already makes aggregate judgments ("are there too many execute agents?") — now the data matches the query.

## Data structures

- `ActiveSummary` — `Total int`, `ByTaskKey map[string]int`, `ByStatus map[string]int`, `ByRuntime map[string]int`

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — contract migration across multiple consumers requires judgment about what each consumer actually needs

## Verification

### Static
- `go test ./...` — all tests pass
- `ActiveCook` type no longer exists
- `sessionmeta.ReadAll()` not called from mise builder or cmd_status
- Schedule skill SKILL.md references `ActiveSummary`
- Schema docs (`schemadoc/specs.go`) reflect new field structure

### Runtime
- Integration test: dispatch 3 sessions with different task keys, verify ActiveSummary counts are correct
- Verify mise.json contains `active_summary` instead of `active_cooks`
- Run schedule skill against the new brief format, verify it still makes reasonable scheduling decisions
- `noodle status` CLI command shows correct active count without scanning session dirs

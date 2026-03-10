Back to [[archive/plans/37-skip-prioritize-with-queue/overview]]

# Phase 2: Tests

## Goal

Add fixture tests covering the new bootstrap-skip paths using the existing `TestLoopDirectoryFixtures` harness in `loop/fixture_test.go`. Each fixture is a directory under `loop/testdata/` with state inputs, setup, and an expected.md runtime dump.

## Changes

### Fixture: `existing-queue-skips-schedule` (`loop/testdata/`)

Scenario: queue.json has two real work items from a previous run. Mise brief has a plan. The loop should dispatch the existing items without bootstrapping schedule.

- `state-01/setup.json` -- default setup (no special config)
- `state-01/input.json` -- mise result with one plan, queue.json pre-seeded with two execute items
- `.noodle.toml` -- default routing config
- `expected.md` -- runtime dump asserting: `normal_task_scheduled: true`, `spawn_calls: 2` (both work items), `first_spawn.name` is the first work item (not `"schedule"`), `repair_task_scheduled: false`

Pre-seed mechanism: The fixture's `state-01/` directory includes a `.noodle/queue.json` file containing two items with valid task keys. The `applyStateRuntimeSnapshot` helper in `fixture_test.go` copies `.noodle/` files into the test's runtime dir before the cycle runs.

### Fixture: `existing-queue-filters-stale-schedule` (`loop/testdata/`)

Scenario: queue.json has one real work item and one leftover schedule item from a crashed prior run. The loop should filter out the stale schedule item and dispatch only the real item.

- `state-01/setup.json` -- default setup
- `state-01/input.json` -- mise result with one plan
- Pre-seeded `.noodle/queue.json` with one execute item + one schedule item (`"id":"schedule"`)
- `expected.md` -- runtime dump asserting: `normal_task_scheduled: true`, `spawn_calls: 1`, `first_spawn.name` is the work item (not `"schedule"`)

### Fixture: `empty-queue-with-plans-still-bootstraps` (`loop/testdata/`)

Scenario: queue.json is empty (no items), mise brief has plans. The loop should still bootstrap schedule. This is a regression guard ensuring the skip logic does not break the normal bootstrap path.

- `state-01/input.json` -- mise result with plans, no pre-seeded queue
- `expected.md` -- runtime dump asserting: `first_spawn.name: "schedule"`, `spawn_calls: 1`

This is similar to the existing `empty-state-should-schedule` fixture but serves as an explicit regression test scoped to this change.

### Fixture: `queue-with-items-and-adopted-targets` (`loop/testdata/`)

Scenario: queue.json has two real work items AND `adoptedTargets` is non-empty (a target was adopted from a prior cycle). The outer `len(l.adoptedTargets) == 0` guard is false, so the skip path does not fire — items should dispatch through the normal path.

- `state-01/setup.json` -- setup with one adopted target (e.g. `"adoptedTargets": [{"id": "plan-1"}]`)
- `state-01/input.json` -- mise result with one plan, queue.json pre-seeded with two execute items
- `expected.md` -- runtime dump asserting: `normal_task_scheduled: true`, `spawn_calls: 2`, items dispatch normally (not via skip path), `repair_task_scheduled: false`

### Fixture: `only-stale-schedule-bootstraps-fresh` (`loop/testdata/`)

Scenario: queue.json has a single leftover schedule item (`"id":"schedule"`) and nothing else. No active sessions, no adopted targets. Mise brief has plans. The loop should discard the stale schedule item and bootstrap a fresh schedule queue — NOT dispatch the stale item.

- `state-01/setup.json` -- default setup
- `state-01/input.json` -- mise result with one plan, queue.json pre-seeded with one schedule item only
- `expected.md` -- runtime dump asserting: `first_spawn.name: "schedule"`, `spawn_calls: 1`. The spawned schedule is a fresh bootstrap, not the stale one. Queue file should show a new `GeneratedAt` timestamp.

### Fixture: `queue-with-items-active-sessions-no-skip` (`loop/testdata/`)

Scenario: queue.json has real work items but `activeByID` is non-empty (a session is already running). The outer `len(l.activeByID) == 0` guard is false, so the skip path does not fire — items should dispatch through normal routing.

- `state-01/setup.json` -- setup with one active session (e.g. `"activeSessions": [{"id": "sess-1", "taskKey": "plan-1/target-a"}]`)
- `state-01/input.json` -- mise result with one plan, queue.json pre-seeded with one execute item
- `expected.md` -- runtime dump asserting: `normal_task_scheduled: true`, items dispatch via normal path (skip path not triggered), active session continues

### Unit tests for helpers (`loop/schedule_test.go` or `loop/queue_test.go`)

- `TestHasNonScheduleItems` -- table-driven: empty queue returns false, queue with only schedule item returns false, queue with one real item returns true, queue with schedule + real returns true.
- `TestFilterStaleScheduleItems` -- table-driven: empty queue returns empty, queue with only schedule returns empty, queue with real items returns them unchanged, mixed queue returns only non-schedule items.

## Data structures

No new types. Test fixtures use existing `loopFixtureStateInput`, `loopFixtureSetup`, and `loopFixtureRuntimeDump` structures.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Fixture scaffolding and table-driven tests |

## Verification

```bash
go test ./loop/... -run TestLoopDirectoryFixtures
go test ./loop/... -run TestLoopDirectoryFixtures/queue-with-items-and-adopted-targets
go test ./loop/... -run TestLoopDirectoryFixtures/only-stale-schedule-bootstraps-fresh
go test ./loop/... -run TestLoopDirectoryFixtures/queue-with-items-active-sessions-no-skip
go test ./loop/... -run TestHasNonScheduleItems
go test ./loop/... -run TestFilterStaleScheduleItems
```

Run all loop tests to confirm no regressions:
```bash
go test ./loop/...
```

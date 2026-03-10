Back to [[archive/plans/73-testing-strategy/overview]]

# Phase 4: Expand Go Fixture Coverage

## Goal

Promote shared fixture helpers into the `fixturedir` library so new packages can adopt fixtures without duplicating boilerplate, then add fixtures for the snapshot builder and loop edge cases.

## Changes

### Fixture library improvements

The loop's fixture runner has reusable patterns that are currently private to `loop/fixture_test.go`. Lifting them into `fixturedir` makes it trivial to add fixtures to new packages.

**`internal/testutil/fixturedir/runtime.go` (new)** — `ApplyRuntimeSnapshot(tb, state, runtimeDir)`. Copies `.noodle/*` files from a fixture state directory into a temp runtime dir. Currently duplicated as `applyStateRuntimeSnapshot` in `loop/fixture_test.go`. The snapshot fixture runner needs the same logic.

**`internal/testutil/fixturedir/record.go` (new)** — `WriteSectionToExpected(expectedPath, sectionName, data)`. Reads `expected.md`, replaces (or appends) the named `## Section` with a JSON code block containing `data`, preserves frontmatter and other sections. Currently hand-rolled as `writeRuntimeDumpSection` in `loop/fixture_test.go`. Each new fixture runner needs this exact logic for record mode.

**`loop/fixture_test.go`** — Migrate to use `fixturedir.ApplyRuntimeSnapshot` and `fixturedir.WriteSectionToExpected`. Delete the private copies.

**`scripts/scaffold-fixture.sh`** — Add a `--template` flag: `loop` (default, creates `input.ndjson`), `snapshot` (creates `input.json`), `generic` (creates empty state dir). The section name in `expected.md` is left for the runner to fill via record mode.

### Snapshot fixtures

**`internal/snapshot/testdata/` (new fixtures)** — Each fixture provides a `loop.LoopState` as `input.json` plus the runtime files that `LoadSnapshot` actually reads: `control.ndjson`, `queue-events.ndjson`, and session canonical event files under `.noodle/sessions/`.

New fixture cases:
- `empty-runtime` — Empty LoopState, no runtime files. Verifies zero-value snapshot is well-formed with empty arrays (not nil).
- `active-sessions-with-orders` — LoopState with active sessions mapped to orders. Verifies `active_order_ids` derivation, `session_id` assignment on `active` and `merging` stage statuses (both `stages` and `on_failure` paths).
- `completed-sessions-in-recent` — Completed sessions appear in `recent`, not `active`.
- `pending-review-items` — Sessions parked for review. Verifies `pending_reviews` populated with correct worktree info.
- `feed-events-ordering` — Feed events sourced from `control.ndjson` (steer events) and `queue-events.ndjson`, merged and sorted by timestamp.
- `enrich-active-session` — Active session with canonical events. Verifies `enrichActiveSession` extracts token/context aggregation and current action.
- `recent-history-truncation` — More than 20 recent sessions in LoopState. Verifies only 20 appear in snapshot (cap enforcement).
- `nil-loop-state-fields` — LoopState with nil slices and maps (not empty). Verifies snapshot produces well-formed output with empty arrays, not nil (regression guard for e8b4e66).
- `steer-events-interleaved-with-queue-events` — Both `control.ndjson` and `queue-events.ndjson` contain events with overlapping timestamps. Verifies merged feed is correctly sorted.

**`internal/snapshot/fixture_test.go` (new)** — `TestSnapshotDirectoryFixtures`. For each fixture state: copy `.noodle/` files into temp runtime dir via `fixturedir.ApplyRuntimeSnapshot`, parse `input.json` as `loop.LoopState`, call `LoadSnapshot`, compare to "Expected Snapshot" section. Record mode via `NOODLE_SNAPSHOT_FIXTURE_MODE` env var uses `fixturedir.WriteSectionToExpected`.

### Loop fixtures

**`loop/testdata/` (new fixtures):**

Dispatch & capacity:
- `concurrent-capacity-exhausted` — All cook slots full, new orders queued but not dispatched.
- `merge-queue-backpressure-blocks-dispatch` — Merge queue pending + inflight exceeds `MergeBackpressureThreshold`. Verifies dispatch is suppressed even though cook slots are available.
- `orders-next-promotion-with-inflight-cook` — Schedule writes `orders-next.json` while a cook is inflight. Verifies cycle merges order IDs without duplicates and does not double-dispatch.

Staleness & config:
- `stale-orders-config-mismatch` — Orders reference a config that no longer matches. Verifies staleness detection and re-scheduling.

Control commands:
- `control-reject-during-on-failure` — Order status is `failing`, OnFailure stage is `pending`. Reject command cancels the OnFailure pipeline and marks the order failed.
- `control-request-changes-at-max-concurrency` — Request-changes command arrives when all cook slots are full. Verifies dispatch is deferred until a slot opens.
- `control-requeue-clears-failed-and-retry` — Order is in both `failedTargets` and `pendingRetry`. Requeue clears both, making the order eligible for dispatch again.

Recovery:
- `bootstrap-exhausted-continues-loop` — Bootstrap has failed 3 times (`bootstrapAttempts >= 3`). Verifies loop sets `bootstrapExhausted`, logs warning, and continues without spawning.
- `pending-retry-exhausted-marks-failed` — Retry attempt exceeds `MaxRetries` in `processPendingRetries`. Verifies order is marked failed and removed.
- `adopted-session-merge-conflict-parks` — Adopted (crash-recovered) session completes but merge conflicts. Verifies order parks for review rather than failing immediately.

Reconciliation:
- `reconcile-merging-stage-branch-merged` — Stage stuck in `merging` status, but branch is already merged to HEAD. Reconcile advances the stage to completed.
- `reconcile-merging-stage-missing-metadata` — Stage in `merging` status with empty `merge_worktree` in Extra. Verifies `failMergingStage` is called gracefully.

Note: `worktree-create-name-collision` already covered by `branch-exists-worktree-create-fails`. `adopted-session-completes` already covered by `TestCycleCompletesAdoptedCookFromMetaState`. No duplicates.

## Data Structures

- `fixturedir.ApplyRuntimeSnapshot` — copies `.noodle/` prefixed files from state to runtime dir
- `fixturedir.WriteSectionToExpected` — splice JSON section into expected.md preserving other content
- Snapshot fixture input: `input.json` → `loop.LoopState` + `.noodle/` runtime files
- Snapshot fixture output: "Expected Snapshot" markdown section → `snapshot.Snapshot` JSON

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Fixture creation is mechanical — write state files, run in record mode, verify output |

## Verification

### Static
- `go test ./...` — all tests pass (including loop fixtures using the new shared helpers)
- `go vet ./...`
- `pnpm fixtures:hash` — all fixture hashes current

### Runtime
- Loop fixtures still pass after migrating to shared helpers (no behavior change)
- Run each new snapshot fixture in record mode to capture baseline output
- Switch to check mode and verify it passes
- Intentionally mutate an input file and confirm the hash check fails

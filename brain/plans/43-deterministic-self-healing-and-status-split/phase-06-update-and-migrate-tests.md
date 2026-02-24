Back to [[plans/43-deterministic-self-healing-and-status-split/overview]]

# Phase 6: Update and migrate tests

## Goal

Migrate existing runtime repair fixture tests to the new two-tier model. Remove tests for deleted features (fingerprinting, adoption). Add tests for the new triage flow.

## Changes

### Update fixture tests for simplified features

- **`runtime-repair-adopt-running-session/`** — Update to test the simplified reattach flow (scan for running `repair-runtime-*` sessions, reattach without fingerprint matching).
- **`runtime-repair-adopt-running-session-no-duplicate/`** — Still valid and important. Rename to clarify: tests that a running repair session prevents spawning a duplicate.
- Any fixture that asserts fingerprint values — update to use `scope|message` key instead.

### Update remaining repair fixtures

These fixtures test behaviors that still exist but with simplified internals:
- **`runtime-repair-completed-resumes-queue/`** — Still valid. Update if state shape changed.
- **`runtime-repair-max-attempts/`** — Still valid. Update to use `scope|message`-based tracking instead of fingerprint.
- **`runtime-repair-spawn-fatal/`** — Still valid. Dispatcher failure path unchanged.
- **`runtime-repair-spawn-fatal-by-name/`** — Still valid. Same dispatcher failure path with named skill.
- **`runtime-repair-exited-fatal/`** — Still valid. Session exit detection simplified.
- **`runtime-repair-idempotent-extra-cycles/`** — Still valid.
- **`runtime-repair-inflight-suppresses-queued-work/`** — Still valid.
- **`runtime-repair-malformed-state/`** — May need updating depending on what "malformed state" means post-redesign.
- **`runtime-repair-oops-fallback-custom-routing/`** — Still valid. Custom routing path unchanged.

### Add new triage fixtures

- **Deterministic fix resolves issue** — Malformed queue.json triggers triage, fixer resets it, no agent spawned, loop continues.
- **Deterministic fix insufficient, agent escalates** — Valid-but-invalid queue (e.g. bad task keys) triggers triage, fixer doesn't match, agent spawns.
- **Stale session cleanup** — Meta.json shows "running" but process is dead, fixer marks "exited", no agent.
- **Restart with dead repair session** — Loop starts, finds a stale repair session meta.json (status "running" but tmux session dead). Stale session fixer marks it "exited". The original issue persists, so a fresh repair agent is spawned.
- **Restart with live repair session** — Loop starts, finds a running `repair-runtime-*` session (tmux alive). Reattaches via the simplified adoption path, pauses loop, waits for completion without spawning a duplicate.

### Update fixture test infrastructure

- **`loop/fixture_test.go`** — Update `runtimeRepairNamePattern` if fingerprint-based naming changed. Keep `RunningRuntimeRepairSessionID` setup (adoption fixtures are updated, not deleted).
- Update `runtimeRepairCalls()` helper if the dispatch request shape changed.

### Non-fixture loop tests asserting loop state in queue

These inline tests in `loop/loop_test.go` assert `queue.LoopState` or `queue.Autonomy` directly — they need updating to read from status.json instead:
- `TestCycleIdleWakesWhenPlansAppear` (line ~436) — asserts `queue.LoopState == "idle"`
- `TestCycleStampsLoopStateWhenPaused` (line ~1046) — asserts loop state and autonomy stamped into queue

### Status file in fixtures

All fixtures that previously asserted loop state in queue.json need updating:
- Queue golden files should not contain `active`/`loop_state`/`autonomy`
- Add status.json golden files where loop state assertions are needed

## Data structures

No new types — fixture test infrastructure types only.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Test migration requires judgment about which fixtures to keep/update/delete |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
make fixtures-loop MODE=check
make fixtures-hash MODE=check
```

### Runtime
- All existing passing tests still pass (after migration)
- New triage fixtures pass
- No test references fingerprinting, adoption, or loop state in queue.json

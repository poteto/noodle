Back to [[plans/72-go-structural-cleanup/overview]]

# Phase 4: Decompose Loop struct

## Goal

Break the 39-field `Loop` struct into cohesive sub-structs that group related fields. Methods stay on Loop — sub-structs are organizational, not architectural boundaries. Loop accesses grouped fields through `l.cooks.xxx`, `l.cmds.xxx`, etc.

Per [[principles/foundational-thinking]]: "The right structure makes downstream code obvious."

## Changes

Extract three field-grouping sub-structs from `Loop`:

### `completionBuffer`
**Fields:** `completions`, `completionOverflow`, `completionOverflowMu`, `completionOverflowSaturated`

**New file:** `loop/completion_buffer.go` — struct definition only.

### `cookTracker`
**Fields:** `activeCooksByOrder`, `adoptedTargets`, `adoptedSessions`, `failedTargets`, `pendingReview`, `pendingRetry`

**New file:** `loop/cook_tracker.go` — struct definition only.

### `cmdProcessor`
**Fields:** `processedIDs`, `cmdSeqCounter`, `lastAppliedSeq`

**New file:** `loop/cmd_processor.go` — struct definition only.

### Loop struct update

**`loop/types.go`** — The Loop struct composes the three sub-structs:
```
Loop {
    // Infrastructure
    projectDir, runtimeDir, config, deps, logger

    // Components
    cooks         cookTracker
    cmds          cmdProcessor
    completionBuf completionBuffer

    // Remaining state
    state, registryStale, registry, registryErr, registryFailCount
    bootstrapAttempts, bootstrapExhausted, bootstrapInFlight
    orders, ordersLoaded
    activeSummary, recentHistory, sessionHealth, mergeQueue
    publishedState, lastStatus
    watcherWG, watcherCount, dispatchGeneration

    // Test hooks
    TestFlushBarrier, TestControlAckBarrier
}
```

**`loop/loop.go`** — Update `New()` to initialize sub-components.

**All methods stay on Loop.** They access grouped fields through `l.cooks.activeCooksByOrder`, `l.cmds.lastAppliedSeq`, `l.completionBuf.completions`, etc. No callbacks, interfaces, or back-pointers needed.

**`loop/*.go` and `loop/*_test.go`** — Update all field access sites to go through the sub-struct (e.g., `l.pendingRetry` → `l.cooks.pendingRetry`, `l.lastAppliedSeq` → `l.cmds.lastAppliedSeq`). This includes test files that construct `Loop{}` literals or access fields directly.

## Data structures

Three new unexported structs: `completionBuffer`, `cookTracker`, `cmdProcessor`. No new exported API. All three are initialized in `New()` and accessed through Loop's composed fields.

## Verification

- `go test ./...` — all tests pass (including integration tests)
- `go vet ./...` — no issues
- The Loop struct field count drops from ~39 to ~29
- No exported API changes — only internal reorganization

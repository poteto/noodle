# Loop Test Overlap Matrix

This matrix maps `loop/loop_test.go` cases to `loop/testdata` fixture coverage.
The goal is to prune only true duplicates and keep narrow unit diagnostics.

## Pruned As Fixture-Equivalent

- `TestCycleSchedulesRuntimeRepairForMiseErrors`
  - Fixture coverage: `runtime-repair-adopt-running-session`,
    `runtime-repair-max-attempts`, `runtime-repair-spawn-fatal`,
    `runtime-repair-oops-fallback-custom-routing`
  - Reason: fixture suite already asserts repair scheduling, oops task routing,
    in-flight state transitions, and worktree creation behavior.

- `TestCycleResumesSchedulingAfterRepairCompletion`
  - Fixture coverage: `runtime-repair-completed-resumes-queue`
  - Reason: fixture suite already validates two-cycle behavior where repair
    completion clears in-flight state and queued work resumes.

## Kept As Unit-Only Or Narrow Diagnostics

- Name/format helpers (`TestCookBaseNameIncludesIDAndShortTitle`,
  `TestCookBaseNameFallsBackToIDWithoutTitle`, `TestTmuxSessionNameMatchesSanitizedLength`)
- Pure helpers and IO checks (`TestCopyVerdictToRuntime`,
  `TestReadQualityVerdictFile`, `TestReadQualityVerdictFileMissing`,
  `TestReadSessionTargetAcceptsRichIDs`, `TestReadSessionTargetDetectsPrioritizePrompt`)
- Control and cancellation behavior (`TestProcessControlCommandsPauseAndAck`,
  `TestRunQualityCancelsSpawnedSessionOnContextDone`)
- Adopted-session and queue edge cases without direct fixture equivalents

## Partial Overlap (Intentionally Kept For Now)

- Spawn and worktree error handling tests near the top of `loop/loop_test.go`
  have related fixture coverage but still provide focused assertions on
  specific call-level invariants. Keep until fixture assertions are expanded.

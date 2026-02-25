# Loop Test Overlap Matrix

This matrix maps `loop/loop_test.go` cases to `loop/testdata` fixture coverage.
The goal is to prune only true duplicates and keep narrow unit diagnostics.

## Pruned — Runtime Repair Removed

The runtime repair system was deleted in plan 38 phase 3. All repair-specific
tests and fixtures were removed:

- `TestCycleSchedulesRuntimeRepairForMiseErrors` — deleted (repair no longer exists)
- `TestCycleResumesSchedulingAfterRepairCompletion` — deleted (repair no longer exists)
- All `runtime-repair-*` fixtures — deleted
- All `missing-sync-*` fixtures — deleted (depended on repair)
- `schedule-exited-without-complete-should-fail` — deleted (depended on repair session status)
- `schedule-failed-retries-should-surface-*` — deleted (tested repair retry chains)

## Kept As Unit-Only Or Narrow Diagnostics

- Name/format helpers (`TestCookBaseNameIncludesIDAndShortTitle`,
  `TestCookBaseNameFallsBackToIDWithoutTitle`, `TestTmuxSessionNameMatchesSanitizedLength`)
- Pure helpers and IO checks (`TestCopyVerdictToRuntime`,
  `TestReadQualityVerdictFile`, `TestReadQualityVerdictFileMissing`,
  `TestReadSessionTargetAcceptsRichIDs`, `TestReadSessionTargetDetectsSchedulePrompt`)
- Control and cancellation behavior (`TestProcessControlCommandsPauseAndAck`,
  `TestRunQualityCancelsSpawnedSessionOnContextDone`)
- Adopted-session and queue edge cases without direct fixture equivalents

## Partial Overlap (Intentionally Kept For Now)

- Spawn and worktree error handling tests near the top of `loop/loop_test.go`
  have related fixture coverage but still provide focused assertions on
  specific call-level invariants. Keep until fixture assertions are expanded.

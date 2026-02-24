Back to [[archived_plans/39-charmbracelet-v2-upgrade/overview]]

# Phase 6: Phase 6 - Verification and cleanup

**Routing:** `codex` / `gpt-5.3-codex` — verification-heavy execution with mechanical cleanup

## Goal

Prove the full migration works end-to-end, remove leftover v1 references, and leave the repo in a clean post-upgrade state.

## Changes

- Remove residual v1-era symbols/imports discovered during previous phases.
- Update any documentation comments or internal notes that still reference v1 API names.
- Ensure tests cover migrated behavior rather than preserving obsolete v1 assumptions.
- Add final regression assertions for key and control-command flows if phase work revealed uncovered paths.
- Record follow-up tasks only for truly out-of-scope issues discovered during verification.

## Data structures

- No new product data structures.
- Verification artifacts are test expectations and runtime observations confirming the v2 contract.

## Verification

Static:
- `go test ./...`
- `go vet ./...`
- `sh scripts/lint-arch.sh`

Runtime:
- Capture `/tmp/noodle-v2-capture/phase-06.txt` using the same tmux script as baseline.
- Compare normalized invariant markers with baseline (`Feed|Queue|Brain|Config|New Task`) and require no marker diffs.
- Verify transcript has no crash/fatal text (`panic:`, `fatal error`).
- Run required interaction tests explicitly:
  - `go test ./tui -run 'TestTabSwitching|TestSteerMentionSelectionInsertsTarget|TestTaskEditorTabCyclesFields|TestTaskEditorSubmitWritesEnqueue|TestDoubleCtrlCQuits|TestQueueTabKeyNavigationMovesCursor|TestQueueTabSelectedSessionIDTracksCursor'`
- Run performance telemetry gate and compare against `/tmp/noodle-v2-capture/perf-baseline.txt` thresholds:
  - `go test ./tui -run 'TestModelViewRenderLatencyBudget|TestQueueTabRenderLatencyBudget' -count=1 -v | tee /tmp/noodle-v2-capture/perf-phase-06.txt`

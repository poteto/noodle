Back to [[archive/plans/39-charmbracelet-v2-upgrade/overview]]

# Phase 4: Phase 4 - Bubbles component API migration

**Routing:** `codex` / `gpt-5.3-codex` — focused component migration with limited surface area

## Goal

Migrate Bubbles component usage to v2 APIs, centered on the queue table integration and any transitive style/keymap changes.

## Changes

- Refactor `tui/queue.go` for Bubbles v2:
  - Ensure `table` construction, sizing, cursor movement, and style setup match v2 contracts.
  - Apply getter/setter method usage where v2 replaced direct width/height fields.
- Audit all Bubbles usage in repo (`table`, and any indirect imports) for v2-only symbols and remove v1 leftovers.
- Confirm Bubbles + Bubble Tea message interop remains correct in queue interactions.

## Data structures

- `QueueTab` table model state remains the core structure, but now backed by v2 component contracts.
- Queue row and status model unchanged intentionally to preserve UX while API surface changes under it.

## Verification

Static:
- `go test ./tui/...`
- `go test ./...`
- `go vet ./...`
- `sh scripts/lint-arch.sh`

Runtime:
- Capture `/tmp/noodle-v2-capture/phase-04.txt` using the same tmux script as baseline.
- Compare normalized invariant markers with baseline (`Feed|Queue|Brain|Config|New Task`) and require no marker diffs.
- Verify transcript has no crash/fatal text (`panic:`, `fatal error`).
- Run interaction-test gate:
  - `go test ./tui -run 'TestTabSwitching|TestSteerMentionSelectionInsertsTarget|TestTaskEditorTabCyclesFields|TestTaskEditorSubmitWritesEnqueue|TestDoubleCtrlCQuits|TestQueueTabKeyNavigationMovesCursor|TestQueueTabSelectedSessionIDTracksCursor'`
- Run performance telemetry gate and compare against `/tmp/noodle-v2-capture/perf-baseline.txt` thresholds:
  - `go test ./tui -run 'TestModelViewRenderLatencyBudget|TestQueueTabRenderLatencyBudget' -count=1 -v | tee /tmp/noodle-v2-capture/perf-phase-04.txt`

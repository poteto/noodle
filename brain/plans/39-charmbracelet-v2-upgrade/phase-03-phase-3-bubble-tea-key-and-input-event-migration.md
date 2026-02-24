Back to [[plans/39-charmbracelet-v2-upgrade/overview]]

# Phase 3: Phase 3 - Bubble Tea key and input event migration

**Routing:** `codex` / `gpt-5.3-codex` — broad but deterministic event/message rewrites

## Goal

Migrate all key-input handling and related tests from Bubble Tea v1 key semantics to v2 key message types and fields.

## Changes

- Refactor message handling in:
  - `tui/model.go`
  - `tui/task_editor.go`
- Replace `tea.KeyMsg` handling with v2 key message handling (`tea.KeyPressMsg` and any required shared key interfaces).
- Update key field usage where needed (`Type`/`Runes`/old constants to v2 `Code`/`Text`/modifiers).
- Normalize key string checks impacted by v2 (for example `" "` -> `"space"` where string matching is used).
- Update all key-driving tests in `tui/model_test.go` (including test helpers `pressRune`/`pressKey`) to construct v2-compatible key messages.
- Add or upgrade automated key-flow tests for high-risk paths (tab switch, queue navigation, steer mention selection, task-editor submit/cancel) so v2 key semantics are covered in CI.
- Required interaction test coverage in this phase:
  - tab switching: `TestTabSwitching`
  - steer mention selection: `TestSteerMentionSelectionInsertsTarget`
  - task-editor field navigation: `TestTaskEditorTabCyclesFields`
  - task-editor submit command path: `TestTaskEditorSubmitWritesEnqueue`
  - quit confirmation path: `TestDoubleCtrlCQuits`
  - queue key navigation path: `TestQueueTabKeyNavigationMovesCursor`
  - queue selection mapping path: `TestQueueTabSelectedSessionIDTracksCursor`

## Data structures

- Key-event model: internal assumptions about key payloads become v2-native (`Code`, `Text`, modifiers).
- Test helper constructors become the canonical way to generate synthetic key events.

## Verification

Static:
- `go test ./tui/...`
- `go test ./...`
- `go vet ./...`
- `sh scripts/lint-arch.sh`

Runtime:
- Capture `/tmp/noodle-v2-capture/phase-03.txt` using the same tmux script as baseline.
- Compare normalized invariant markers with baseline (`Feed|Queue|Brain|Config|New Task`) and require no marker diffs.
- Verify transcript has no crash/fatal text (`panic:`, `fatal error`).
- Run required interaction tests explicitly:
  - `go test ./tui -run 'TestTabSwitching|TestSteerMentionSelectionInsertsTarget|TestTaskEditorTabCyclesFields|TestTaskEditorSubmitWritesEnqueue|TestDoubleCtrlCQuits|TestQueueTabKeyNavigationMovesCursor|TestQueueTabSelectedSessionIDTracksCursor'`
- Run performance telemetry gate and compare against `/tmp/noodle-v2-capture/perf-baseline.txt` thresholds:
  - `go test ./tui -run 'TestModelViewRenderLatencyBudget|TestQueueTabRenderLatencyBudget' -count=1 -v | tee /tmp/noodle-v2-capture/perf-phase-03.txt`

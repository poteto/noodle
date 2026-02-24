Back to [[archived_plans/39-charmbracelet-v2-upgrade/overview]]

# Phase 2: Phase 2 - Bubble Tea view and program lifecycle migration

**Routing:** `claude` / `claude-opus-4-6` — architecture-sensitive migration of core TUI contract

## Goal

Migrate the core Bubble Tea model/program contract from v1 to v2 by adopting declarative `tea.View` output and removing deprecated imperative program options.

## Changes

- Refactor `tui/model.go`:
  - `View() string` -> `View() tea.View`
  - Build `tea.View` content via `tea.NewView(...)`
  - Set view flags (for example alt-screen behavior) declaratively on the returned view.
- Refactor `cmd_start.go` TUI startup path:
  - Remove obsolete v1 program options in `tea.NewProgram(...)`
  - Keep context/shutdown behavior intact while adopting v2-compatible startup.
- Audit for renamed/removed Bubble Tea APIs (for example `tea.Sequentially` / `tea.WindowSize`) and apply direct v2 replacements if encountered.

## Data structures

- `Model.View` return type becomes `tea.View`, the foundational contract for all future rendering behavior.
- Startup lifecycle contract in `runTUI` reflects v2 program construction assumptions.

## Verification

Static:
- `go test ./tui/...`
- `go test ./...`
- `go vet ./...`
- `sh scripts/lint-arch.sh`

Runtime:
- Capture `/tmp/noodle-v2-capture/phase-02.txt` using the same tmux script as baseline.
- Compare normalized invariant markers with baseline (`Feed|Queue|Brain|Config|New Task`) and require no marker diffs.
- Verify transcript has no crash/fatal text (`panic:`, `fatal error`).
- Run interaction-test gate:
  - `go test ./tui -run 'TestTabSwitching|TestSteerMentionSelectionInsertsTarget|TestTaskEditorTabCyclesFields|TestTaskEditorSubmitWritesEnqueue|TestDoubleCtrlCQuits|TestQueueTabKeyNavigationMovesCursor|TestQueueTabSelectedSessionIDTracksCursor'`
- Run performance telemetry gate and compare against `/tmp/noodle-v2-capture/perf-baseline.txt` thresholds:
  - `go test ./tui -run 'TestModelViewRenderLatencyBudget|TestQueueTabRenderLatencyBudget' -count=1 -v | tee /tmp/noodle-v2-capture/perf-phase-02.txt`

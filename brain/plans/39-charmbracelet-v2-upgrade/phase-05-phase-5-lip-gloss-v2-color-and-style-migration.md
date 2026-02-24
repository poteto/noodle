Back to [[plans/39-charmbracelet-v2-upgrade/overview]]

# Phase 5: Phase 5 - Lip Gloss v2 color and style migration

**Routing:** `claude` / `claude-opus-4-6` — requires judgment on color typing and style-system redesign-from-first-principles

## Goal

Migrate Lip Gloss usage from v1 color types to v2 color APIs while preserving current visual design and component boundaries.

## Changes

- Refactor color-typed theme/style structures in:
  - `tui/styles.go`
  - `tui/components/theme.go`
  - `tui/components/card.go`
  - `tui/components/pill.go`
  - `tui/feed_item.go`
  - `tui/verdict.go`
  - `tui/queue.go` and other call sites using `lipgloss.Color` as a type
- Replace any remaining v1 assumptions:
  - `lipgloss.Color` as a concrete type
  - removed renderer-centric APIs (if present)
  - removed adaptive-color APIs (if present)
- Keep output strategy simple: Bubble Tea v2 handles terminal rendering/downsampling for TUI output; avoid introducing compat/global writer complexity unless required by tests.

## Data structures

- Theme/color structs move to `image/color.Color`-compatible fields.
- Shared style-building functions become the single color-boundary layer for TUI components.

## Verification

Static:
- `go test ./tui/...`
- `go test ./...`
- `go vet ./...`
- `sh scripts/lint-arch.sh`

Runtime:
- Capture `/tmp/noodle-v2-capture/phase-05.txt` using the same tmux script as baseline.
- Compare normalized invariant markers with baseline (`Feed|Queue|Brain|Config|New Task`) and require no marker diffs.
- Verify transcript has no crash/fatal text (`panic:`, `fatal error`).
- Run interaction-test gate:
  - `go test ./tui -run 'TestTabSwitching|TestSteerMentionSelectionInsertsTarget|TestTaskEditorTabCyclesFields|TestTaskEditorSubmitWritesEnqueue|TestDoubleCtrlCQuits|TestQueueTabKeyNavigationMovesCursor|TestQueueTabSelectedSessionIDTracksCursor'`
- Run performance telemetry gate and compare against `/tmp/noodle-v2-capture/perf-baseline.txt` thresholds:
  - `go test ./tui -run 'TestModelViewRenderLatencyBudget|TestQueueTabRenderLatencyBudget' -count=1 -v | tee /tmp/noodle-v2-capture/perf-phase-05.txt`

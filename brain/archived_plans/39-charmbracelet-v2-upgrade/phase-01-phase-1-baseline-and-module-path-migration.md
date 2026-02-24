Back to [[archived_plans/39-charmbracelet-v2-upgrade/overview]]

# Phase 1: Phase 1 - Baseline and module path migration

**Routing:** `codex` / `gpt-5.3-codex` — mechanical import and module rewrites

## Goal

Move the codebase onto the v2 module paths and establish a real compiling baseline before deeper semantic behavior changes.

## Changes

- Update `go.mod` and `go.sum` to:
  - `charm.land/bubbletea/v2`
  - `charm.land/bubbles/v2`
  - `charm.land/lipgloss/v2`
- Rewrite imports across all TUI and startup files (`cmd_start.go`, `tui/*.go`, `tui/components/*.go`, tests) to v2 module paths.
- Apply compile-unblocking API renames that are required to restore buildability immediately after module/path migration.
- Keep semantic interaction changes (key behavior details, runtime flow checks) for later phases, but do not end Phase 1 with a broken build.

## Data structures

- Dependency graph (`go.mod` / `go.sum`) becomes the source of truth for v2 modules.
- Migration inventory for remaining semantic follow-up work (grouped by Bubble Tea, Bubbles, Lip Gloss) to drive follow-on phases.

## Verification

Static:
- `go test ./tui/...`
- `go test ./...`
- `go vet ./...`
- `sh scripts/lint-arch.sh`

Runtime:
- Before making code changes in this phase, create the baseline capture at `/tmp/noodle-v2-capture/baseline.txt` using the overview protocol.
- At end of phase, capture `/tmp/noodle-v2-capture/phase-01.txt` with the same scripted tmux interaction.
- Compare normalized invariant markers with baseline (`Feed|Queue|Brain|Config|New Task`) and require no marker diffs.
- Verify transcript has no crash/fatal text (`panic:`, `fatal error`).
- Run interaction-test gate:
  - `go test ./tui -run 'TestTabSwitching|TestSteerMentionSelectionInsertsTarget|TestTaskEditorTabCyclesFields|TestTaskEditorSubmitWritesEnqueue|TestDoubleCtrlCQuits|TestQueueTabKeyNavigationMovesCursor|TestQueueTabSelectedSessionIDTracksCursor'`
- Capture performance telemetry baseline and validate latency budgets:
  - `go test ./tui -run 'TestModelViewRenderLatencyBudget|TestQueueTabRenderLatencyBudget' -count=1 -v | tee /tmp/noodle-v2-capture/perf-baseline.txt`

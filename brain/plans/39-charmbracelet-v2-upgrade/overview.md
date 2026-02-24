---
id: 39
created: 2026-02-24
status: ready
---

# Charmbracelet V2 Upgrade

## Context

Noodle's TUI currently depends on Charmbracelet v1-era APIs:
- `github.com/charmbracelet/bubbletea` (`v1.3.10`)
- `github.com/charmbracelet/bubbles` (`v0.21.x` pre-v2 snapshot)
- `github.com/charmbracelet/lipgloss` (`v1.1.x` pre-v2 snapshot)

The upstream migration guides introduce breaking changes across all three packages and explicitly require upgrading them together:
- Bubble Tea: `View() tea.View`, `tea.KeyPressMsg`, declarative view fields
- Lip Gloss: `charm.land/lipgloss/v2`, `image/color.Color`-oriented APIs, renderer removal
- Bubbles: `charm.land/bubbles/v2`, key/type/style API changes tied to Bubble Tea v2 and Lip Gloss v2

This plan coordinates one migration wave so the TUI stays coherent and we avoid piecemeal breakage.

## Migration Guides

- Bubble Tea v2: https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md
- Lip Gloss v2: https://github.com/charmbracelet/lipgloss/blob/main/UPGRADE_GUIDE_V2.md
- Bubbles v2: https://github.com/charmbracelet/bubbles/blob/main/UPGRADE_GUIDE_V2.md

## Scope

**In:**
- Upgrade module imports and dependencies to the `charm.land/*/v2` module paths
- Migrate Bubble Tea program/model code in `cmd_start.go`, `tui/model.go`, and `tui/task_editor.go`
- Migrate Bubbles table integration in `tui/queue.go`
- Migrate Lip Gloss color typing and style plumbing across `tui/` and `tui/components/`
- Update TUI-focused tests (`tui/model_test.go`, `tui/components/components_test.go`) for v2 message/types behavior

**Out:**
- New TUI feature work unrelated to compatibility
- Theme redesign, layout redesign, or interaction redesign
- Backward-compatibility shims for v1 APIs (explicitly out by project policy)
- Non-TUI dependency upgrades

## Constraints

- Cross-platform behavior must remain stable on macOS/Linux/Windows.
- No dual-path compatibility layer: migrate callers, then delete legacy v1 usage.
- Follow existing TUI architecture rules from `bubbletea-tui` skill (main model routes messages; components stay dumb).
- Keep changes small and phaseable, but each phase must end in a verifiable working state.

Alternatives considered:
1. Big-bang single PR touching everything at once. Rejected: high risk and difficult rollback/debugging.
2. Add compatibility wrappers around v1 APIs. Rejected: conflicts with "no backward compatibility by default" and adds dead weight.
3. Chosen: phased migration inside one plan, ordered by dependency boundaries (imports/contracts first, then event semantics, then style/type cleanup, then end-to-end verification).

## Applicable Skills

- `bubbletea-tui` — preserve component/rendering architecture and TUI ergonomics during migration
- `go-best-practices` — keep API boundary changes explicit and testable in Go

## Phases

- [[plans/39-charmbracelet-v2-upgrade/phase-01-phase-1-baseline-and-module-path-migration]]
- [[plans/39-charmbracelet-v2-upgrade/phase-02-phase-2-bubble-tea-view-and-program-lifecycle-migration]]
- [[plans/39-charmbracelet-v2-upgrade/phase-03-phase-3-bubble-tea-key-and-input-event-migration]]
- [[plans/39-charmbracelet-v2-upgrade/phase-04-phase-4-bubbles-component-api-migration]]
- [[plans/39-charmbracelet-v2-upgrade/phase-05-phase-5-lip-gloss-v2-color-and-style-migration]]
- [[plans/39-charmbracelet-v2-upgrade/phase-06-phase-6-verification-and-cleanup]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Runtime verification evidence (tmux baseline + deterministic invariants):
1. Before any phase work, create baseline transcript in `/tmp`:
```bash
mkdir -p /tmp/noodle-v2-capture
tmux new-session -d -s noodle-v2-capture 'cd /Users/lauren/code/noodle && go run . start'
sleep 2
tmux send-keys -t noodle-v2-capture:0.0 1 2 3 4 j k n Escape C-c C-c
sleep 1
tmux capture-pane -pt noodle-v2-capture:0.0 -S -300 > /tmp/noodle-v2-capture/baseline.txt
tmux kill-session -t noodle-v2-capture
```
2. At the end of each phase, run the same scripted tmux interaction and save to `/tmp/noodle-v2-capture/phase-0X.txt`.
3. Compare normalized invariant markers, not byte-for-byte output:
```bash
perl -pe 's/\e\[[0-9;?]*[ -\\/]*[@-~]//g' /tmp/noodle-v2-capture/baseline.txt > /tmp/noodle-v2-capture/baseline.norm.txt
perl -pe 's/\e\[[0-9;?]*[ -\\/]*[@-~]//g' /tmp/noodle-v2-capture/phase-0X.txt > /tmp/noodle-v2-capture/phase-0X.norm.txt
rg -o 'Feed|Queue|Brain|Config|New Task' /tmp/noodle-v2-capture/baseline.norm.txt | sort -u > /tmp/noodle-v2-capture/baseline.markers.txt
rg -o 'Feed|Queue|Brain|Config|New Task' /tmp/noodle-v2-capture/phase-0X.norm.txt | sort -u > /tmp/noodle-v2-capture/phase-0X.markers.txt
diff -u /tmp/noodle-v2-capture/baseline.markers.txt /tmp/noodle-v2-capture/phase-0X.markers.txt
```
4. Invariant acceptance rule: no marker differences are allowed, and transcript must not contain crash/fatal text (`panic:`, `fatal error`).
5. Interaction-test gate (run every phase): `go test ./tui -run 'TestTabSwitching|TestSteerMentionSelectionInsertsTarget|TestTaskEditorTabCyclesFields|TestTaskEditorSubmitWritesEnqueue|TestDoubleCtrlCQuits|TestQueueTabKeyNavigationMovesCursor|TestQueueTabSelectedSessionIDTracksCursor'`
6. Performance telemetry gate (run every phase):
```bash
go test ./tui -run 'TestModelViewRenderLatencyBudget|TestQueueTabRenderLatencyBudget' -count=1 -v | tee /tmp/noodle-v2-capture/perf-phase-0X.txt
```
Acceptance rule: latency budgets must pass and phase telemetry must not exceed baseline telemetry thresholds captured at migration start.

Back to [[plans/42-requires-approval-gate/overview]]

# Phase 4: Simplify Autonomy Config

## Goal

Simplify the 3-mode autonomy dial to 2 modes. The `review` mode ("quality review runs, auto-merge on APPROVE") no longer makes sense because there's no hardcoded quality review step. Replace with:

- `auto` — Auto-merge on success. Per-skill `permissions.merge` is respected.
- `approve` — Everything requires human approval regardless of per-skill setting.

The `auto` mode is the new default. Skills with `permissions: { merge: false }` still park for human review even in `auto` mode. The `approve` mode is a global override for when the user wants to review everything.

## Changes

### `config/config.go`

- Delete `AutonomyFull`, `AutonomyReview` constants
- Two modes only:
  - `AutonomyAuto = "auto"` (replaces both `full` and `review`)
  - `AutonomyApprove = "approve"` (unchanged)
- Default: `AutonomyAuto`
- Delete `ReviewEnabled()` method — no longer meaningful
- Update `PendingApproval()` — returns true when `Autonomy == AutonomyApprove`
- Update `validateParsedValues` to accept only `auto` and `approve`, reject old values (`full`, `review`)

### `startup/firstrun.go`

Update the scaffolded `.noodle.toml` template — currently writes `autonomy = "review"`. Change to `autonomy = "auto"`.

### `tui/model_snapshot.go`

Update the autonomy default (around line 96) — currently defaults to `review`, change to `auto`.

### `tui/feed.go`

Update the autonomy special-casing (around line 133) that checks for `full` mode.

### `internal/schemadoc/specs.go`

Update the autonomy field documentation (around line 122) — currently describes three modes, update to two.

### `generate/skill_noodle.go`

Update the `fieldDescriptions` entry for `"autonomy"` — currently says `"full, review, or approve"`, change to `"auto or approve"` (around line 35).

### `config/config_test.go`

- Remove `TestLegacyReviewEnabledTrueMigratesToReview` / `False`
- Update `TestPendingApprovalHelper` to test only `auto` vs `approve`
- Remove `TestReviewEnabledHelper`
- Add test that old values (`full`, `review`) are rejected by validation

### `startup/firstrun_test.go`

Update test that asserts `autonomy = "review"` in scaffolded config — change to `autonomy = "auto"`.

### `loop/loop_test.go`

Update autonomy-related tests (around line 1008) that reference old mode values.

### `tui/config_tab.go`

- Update the autonomy dial to show 2 modes instead of 3
- Labels: "Auto" / "Approve"

### `loop/control.go` — `controlAutonomy`

- Accept only `auto` and `approve` values

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — config and behavioral changes need judgment.

## Verification

```sh
go test ./config/... ./loop/... ./tui/... ./startup/...
# Launch TUI and verify autonomy dial shows 2 modes
```

Critical test matrix for `handleCompletion` (may be covered by Phase 2 tests, but verify):
- `auto` + `CanMerge=true` -> auto-merge
- `auto` + `CanMerge=false` -> park
- `approve` + `CanMerge=true` -> park (global override)
- `approve` + `CanMerge=false` -> park

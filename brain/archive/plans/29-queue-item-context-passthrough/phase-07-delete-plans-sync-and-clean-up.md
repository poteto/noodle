Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 7 — Delete plans-sync and clean up

## Goal

Remove the plans adapter infrastructure from Noodle's own adapter and clean up stale references. The `plan` Go package stays — it's used by the `noodle plan` CLI (`cmd_plan.go`).

## Changes

**Noodle's own adapter (`.noodle/adapters/main.go`):**
- Delete `plans-sync` command handler — plans are now surfaced through backlog items' `plan` field
- Remove `plans-sync` from the command router

**Go code cleanup:**
- Remove any plan-related types from `adapter/types.go` that are no longer referenced (e.g., `PlanSummary` if it was adapter-specific)
- Do NOT delete the `plan` package — `cmd_plan.go` depends on `plan.Create`, `plan.Activate`, `plan.Done`, `plan.PhaseAdd`, `plan.ReadAll`

**Config:**
- The plans adapter is not in the default config (`config/config.go` defaults to `backlog` only), so no config change is needed
- If `.noodle.toml` has a plans adapter entry, it becomes a no-op — document this in migration notes

**Documentation:**
- Update `.agents/skills/noodle/SKILL.md` — remove references to the plans adapter, document that plans are surfaced through backlog items
- Update `.agents/skills/noodle/references/adapters.md` — remove `PlanItem` references from sync output description (line 24), update to reflect that plans are optional on backlog items
- Update `generate/skill_noodle.go` if it references plans adapter (run `go generate` after)

**Tests/fixtures/docs:**
- Update adapter fixtures and tests that reference plans-sync
- Update `internal/schemadoc/specs.go` and `internal/schemadoc/render_test.go` if they reference plan adapter types

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.4` |

## Verification

### Static
- `go build ./...` and `go test ./...` pass
- `grep -rn "plans-sync" --include="*.go"` returns no hits outside archived/test fixtures
- `plan` package is still importable and `noodle plan` CLI works

### Runtime
- `noodle start` works end-to-end with only a backlog adapter
- `noodle plan create` / `noodle plan phase-add` still work

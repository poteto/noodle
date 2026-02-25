Back to [[archived_plans/38-resilient-skill-resolution/overview]]

# Phase 2: Non-fatal skill resolution at dispatch

**Routing:** `claude` / `claude-opus-4-6` — judgment call on error handling semantics, affects all dispatch paths

## Goal

Change skill resolution from hard-fail to soft-fail. When a skill can't be found, the dispatcher warns and proceeds without methodology guidance rather than killing the session. The agent still gets the preamble, task prompt, and context — it just doesn't get the specialized skill instructions.

The execute domain skill path (`loadExecuteBundle` lines 105-112) already does this correctly — use it as the model.

## Changes

**Add sentinel error to resolver:**
- `skill/resolver.go` — add `var ErrNotFound = errors.New("skill not found")`. Change `Resolve()` to return `fmt.Errorf("skill %q: %w", name, ErrNotFound)` instead of a plain formatted string. This gives callers a typed error to match on via `errors.Is()`, avoiding brittle string matching.

**Soft-fail in skill bundle loading:**
- `dispatcher/skill_bundle.go` — change `loadSkillBundle()`: when `errors.Is(err, skill.ErrNotFound)`, return an empty `loadedSkill` with the warning appended to the existing `Warnings` field (already present on `loadedSkill`). Other resolution errors (filesystem failures, parse errors) remain hard errors.
- `dispatcher/tmux_dispatcher.go` — no interface change needed. The `loadedSkill.Warnings` are already wired into session events.
- `dispatcher/sprites_dispatcher.go` — same treatment as tmux dispatcher.

## Data structures

- `skill.ErrNotFound` — new sentinel error
- `loadedSkill.Warnings` — already exists, no change needed

## Verification

```bash
go test ./skill/... && go test ./dispatcher/... && go vet ./...
```

Test: dispatch a request with a non-existent skill name. Verify session starts successfully with empty methodology. Verify warning appears in session events. Test that a resolver error that is NOT "not found" (e.g., permission denied reading SKILL.md) still hard-fails.

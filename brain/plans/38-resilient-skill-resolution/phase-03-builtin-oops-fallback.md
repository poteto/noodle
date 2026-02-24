Back to [[plans/38-resilient-skill-resolution/overview]]

# Phase 3: Built-in oops fallback with user override

**Routing:** `claude` / `claude-opus-4-6` — prompt engineering, dispatch path design

## Goal

Runtime repair should always be possible, even with zero user-provided skills. Ship a hardcoded oops prompt derived from the noodle repo's own `oops` and `debugging` skills. User-provided oops skill takes precedence when present.

## Changes

**Add `SystemPrompt` field to dispatch request:**
- `dispatcher/types.go` (or wherever `DispatchRequest` lives) — add `SystemPrompt string` field. Precedence rule: if `SystemPrompt` is non-empty, use it directly and skip skill resolution — regardless of whether `Skill` is also set. This makes the caller's intent unambiguous: setting `SystemPrompt` means "I already have the prompt, don't resolve anything." Callers should not set both; if they do, `SystemPrompt` wins.
- `dispatcher/tmux_dispatcher.go` — check `req.SystemPrompt` before calling `loadSkillBundle()`. If set, construct `loadedSkill` from it directly.
- `dispatcher/sprites_dispatcher.go` — same treatment.

**Hardcode oops prompt:**
- New file: `loop/builtin_oops.go` — contains the built-in oops system prompt as a Go string constant. Derived from the current `.agents/skills/oops/SKILL.md` and `.agents/skills/debugging/SKILL.md` content, focused on noodle-specific failure modes: file format issues, stale state, config problems, skill resolution failures.

**Change runtime repair dispatch:**
- `loop/runtime_repair.go` — rewrite `runtimeRepairSkill()` and `spawnRuntimeRepair()`:
  1. Try `resolver.Resolve("oops")` — if user has an oops skill, use it (set `req.Skill`)
  2. If not found, use built-in prompt (set `req.SystemPrompt`, leave `req.Skill` empty)
  3. Never return `""` — there's always a fallback
- Delete the `runtimeRepairSkill() string` method — replace with a method that returns `(skill string, systemPrompt string)`
- `ensureRuntimeRepair()` — remove the "no repair skill" early-return error path (lines 58-60). Repair is always possible now.

## Data structures

- `DispatchRequest.SystemPrompt string` — new field; when set, takes precedence over `Skill`
- `builtinOopsPrompt` — string constant in `loop/builtin_oops.go`

## Verification

```bash
go test ./loop/... && go test ./dispatcher/... && go vet ./...
```

Tests:
- Dispatch with `SystemPrompt` set: verify skill resolution is skipped, prompt is used directly
- Dispatch with both `SystemPrompt` and `Skill` set: verify `SystemPrompt` wins
- Dispatch with only `Skill` set: verify normal resolution (unchanged behavior)
- Remove oops and debugging skills from `.agents/skills/`. Write malformed `queue.json` (`echo '{"items":[{"id":"x","task_key":"nonexistent"}]}' > .noodle/queue.json`). Verify repair session spawns with built-in prompt.
- Add an oops skill back — verify user skill takes precedence over built-in.

Back to [[plans/59-subtract-go-logic-and-resilience/overview]]

# Phase 6: Wire bootstrap skill dispatch and fix exhaustion

Covers: #59 (wiring + exhaustion fix)

## Goal

Replace the hardcoded bootstrap prompt with skill file loading. Fix the silent exhaustion behavior — when bootstrap fails 3 times, log an actionable diagnostic instead of silently giving up.

## Changes

- `loop/builtin_bootstrap.go` — Delete `bootstrapPromptTemplate` constant and `buildBootstrapPrompt()` package-level function. Replace with skill resolver lookup: resolve "bootstrap" skill via `l.skillResolver.Resolve("bootstrap")`, read its content, strip frontmatter, substitute `{{history_dirs}}`, and pass as `req.SystemPrompt` on the `dispatcher.DispatchRequest`. **Do NOT use `req.Skill = "bootstrap"`** — `loadSkillBundle` doesn't do template substitution and treats missing skills as warning-only, which wouldn't satisfy the "actionable error on missing skill" requirement. If the bootstrap skill is missing from the resolver, log an error with actionable next steps ("bootstrap skill not found — create .agents/skills/bootstrap/SKILL.md or run noodle init") and return an error instead of silently continuing.
- `loop/schedule.go` (~lines 109-115) — Replace the silent `return nil` on `bootstrapExhausted` with a logged warning: "bootstrap exhausted after N attempts — create .agents/skills/schedule/SKILL.md manually or check bootstrap skill output for errors". Emit a feed event so it's visible in the web UI.
- `loop/cook.go` (~lines 192-195) — On bootstrap failure, log the failure reason (not just increment counter).
- `loop/bootstrap_test.go` — Update all affected tests:
  - `TestBootstrapPromptContainsHistoryDirForProvider` (~line 339) — calls deleted `buildBootstrapPrompt`. Rewrite to test skill-file-based prompt loading with `{{history_dirs}}` substitution.
  - `TestBootstrapSessionUsesSystemPromptNotSkill` (~line 76) and `TestSystemPromptFieldOnDispatchRequest` (~line 122) — may need bootstrap skill fixture in temp dir since tests create temp projects without `.agents/skills/bootstrap/SKILL.md`.
  - Other bootstrap lifecycle tests — add bootstrap skill fixture to test setup.

## Data structures

No new types. `bootstrapPromptTemplate` constant and `buildBootstrapPrompt()` function deleted.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical wiring — delete constant, add resolver call, add logging |

## Verification

### Static
- `go test ./loop/...` passes
- `go vet ./...` clean
- Grep for `bootstrapPromptTemplate` — zero hits
- New test: bootstrap dispatch loads skill content from resolver
- New test: bootstrap exhaustion logs warning (check logger output)
- New test: missing bootstrap skill logs actionable error

### Runtime
- Delete schedule skill, run `noodle start --once` → bootstrap dispatches using skill file content
- Let bootstrap fail 3 times → verify warning logged with actionable steps (not silent)

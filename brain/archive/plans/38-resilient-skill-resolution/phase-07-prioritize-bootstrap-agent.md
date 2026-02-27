Back to [[archive/plans/38-resilient-skill-resolution/overview]]

# Phase 7: Prioritize bootstrap agent

**Routing:** `claude` / `claude-opus-4-6` — prompt design, workflow inspection logic, the capstone feature

## Goal

When the loop needs to bootstrap prioritization but no prioritize skill exists, spawn a special agent that:
1. Inspects the user's `.claude/` or `.codex/` directory (based on configured provider) to understand their workflow from conversation history
2. Creates a prioritize skill tailored to the user's project
3. Commits the skill file
4. Exits — does NOT perform prioritization in the same session

The next cycle picks up the new skill and runs real prioritization.

## Changes

**Bootstrap prompt:**
- New file: `loop/builtin_bootstrap.go` — contains the bootstrap agent system prompt as a Go string constant (embedded, no remote fetch). The prompt instructs the agent to:
  - Read `.claude/` or `.codex/` conversation history files
  - Understand the project's domain, workflow patterns, and priorities
  - Read existing `brain/todos.md` and `brain/plans/` to understand backlog shape
  - Reference a bundled example of a prioritize skill (embedded as a string constant in the same file)
  - Create `.agents/skills/prioritize/SKILL.md` with appropriate frontmatter (`noodle: { schedule: "..." }`)
  - Check if the file already exists first — if so, exit immediately (idempotency)
  - `git add` and `git commit` the new skill
  - Exit

**Add `SystemPrompt` field to dispatch request:**
- `dispatcher/types.go` — add `SystemPrompt string` field to `DispatchRequest`. When set, use it directly and skip skill resolution. This lets the bootstrap agent dispatch without a skill file existing yet.
- `dispatcher/tmux_dispatcher.go` — check `req.SystemPrompt` before calling `loadSkillBundle()`. If set, construct `loadedSkill` from it directly.
- `dispatcher/sprites_dispatcher.go` — same treatment.

**Change prioritize bootstrap flow:**
- `loop/prioritize.go` — in the prioritize dispatch path, before building the dispatch request:
  1. Call `ensureSkillFresh("prioritize")` (phase 6)
  2. If found → dispatch normally (current behavior)
  3. If not found → dispatch bootstrap agent using `DispatchRequest.SystemPrompt`
  4. Tag the session name with a `bootstrap-` prefix so the loop can distinguish it from real prioritize sessions

**Backoff and idempotency:**
- Track bootstrap attempts with a simple `bootstrapAttempts int` counter on Loop. Max 3 attempts.
- After 3 failed bootstraps: mark as exhausted via `bootstrapExhausted bool` on Loop. The loop continues running — it can still dispatch existing queue items, just can't create new prioritize cycles. When bootstrap is exhausted, the "empty queue" check skips the prioritize bootstrap and returns nil (no error).
- On noodle restart: attempt counter resets (it's in-memory), so bootstrap gets 3 fresh tries.

**Critical: bypass `retryCook` for bootstrap sessions.**
- Bootstrap sessions must NOT flow through the normal `retryCook` path, which hard-errors after retries and would kill the loop. Instead:
  - In `collectCompleted`, check for `bootstrap-` prefix on session name.
  - If bootstrap session failed: increment `bootstrapAttempts` and set `bootstrapExhausted` if at max. Do NOT call `retryCook`. Return nil so the loop continues.
  - If bootstrap session succeeded: trigger `rebuildRegistry()` to pick up the new skill. Write bootstrap completion event.

**Bootstrap completion event:**
- On bootstrap session completion (detected in monitor/reconciliation via `bootstrap-` session name prefix), write a `queue-events.ndjson` entry with `type: "bootstrap"` and a message like `"created prioritize skill from workflow analysis"`. This is the producer contract for phase 8's `"bootstrap"` feed category.

**Adoption on restart:**
- If noodle restarts while a bootstrap session is in tmux, the adoption/reconciliation logic (`reconcile.go`) must recognize `bootstrap-` prefixed sessions. Match against `running|spawning|stuck` states (same set as `runtime_repair.go:146`, not just `running`). If adopted, set the in-flight bootstrap flag to prevent spawning a duplicate.

**Provider-aware history path:**
- The bootstrap prompt should reference the right directory based on `config.Routing.Defaults.Provider`:
  - `"claude"` → inspect `.claude/` directory
  - `"codex"` → inspect `.codex/` directory
  - Both if available

## Data structures

- `builtinBootstrapPrompt` — embedded string constant in `loop/builtin_bootstrap.go`
- `DispatchRequest.SystemPrompt string` — when set, takes precedence over `Skill`
- `Loop.bootstrapAttempts int` — simple counter, reset on restart
- `Loop.bootstrapExhausted bool` — set after 3 failed attempts

## Verification

```bash
go test ./loop/... && go vet ./...
```

Unit tests:
- Missing prioritize skill triggers bootstrap dispatch (not normal prioritize)
- Bootstrap session tagged with `bootstrap-` prefix
- Failed bootstrap does NOT flow through `retryCook` — loop continues
- After 3 failed bootstraps, `bootstrapExhausted` is true and loop skips bootstrap (no thrashing)
- Loop continues dispatching existing queue items when bootstrap is exhausted
- If skill appears between attempts (e.g., user creates it manually), normal prioritize resumes and `bootstrapExhausted` is irrelevant
- Bootstrap prompt includes idempotency check instruction
- Noodle restart with in-flight bootstrap session in `running` state: adopted correctly, no duplicate spawn
- Noodle restart with in-flight bootstrap session in `spawning` state: adopted correctly, no duplicate spawn
- Noodle restart with in-flight bootstrap session in `stuck` state: adopted correctly, no duplicate spawn

Runtime:
```bash
cd "$(scripts/sandbox.sh bare)"
noodle start
# Verify bootstrap agent spawns, creates skill, commits
ls .agents/skills/prioritize/SKILL.md
git log --oneline -3
```

Also test with `scripts/sandbox.sh init` (scaffolding exists but no skills) to verify no duplicate scaffolding during bootstrap.

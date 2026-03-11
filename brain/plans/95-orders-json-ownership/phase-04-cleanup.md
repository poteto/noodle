Back to [[plans/95-orders-json-ownership/overview]]

# Phase 4 — Remove Stale Safety Instructions

## Goal

Remove the "never write orders.json directly" instruction from agent-facing docs. Per encode-lessons-in-structure: if the structure prevents the problem, the instruction is redundant noise.

## Changes

- **`.agents/skills/schedule/SKILL.md`** — remove the prohibition clause
- **`loop/schedule.go`** — check `buildSchedulePrompt()` for similar runtime instructions

## Details

Current text in SKILL.md (line 10):
> The loop atomically promotes `orders-next.json` into `orders.json` — never write `orders.json` directly.

Replace with:
> The loop promotes `orders-next.json` into `orders.json`.

The prohibition is now structurally enforced — agent writes to orders.json are harmless (never read during operation, overwritten by flushState). Keeping the instruction suggests the structure doesn't prevent it, which is misleading.

Also grep for "never write orders" and similar phrases across the codebase to catch any other references.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.3-codex` | Mechanical text edit |

## Verification

### Static
- Grep for "never write orders" across codebase → zero hits

### Runtime
- N/A (documentation change only)

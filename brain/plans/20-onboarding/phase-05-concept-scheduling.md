Back to [[plans/20-onboarding/overview]]

# Phase 5 — Concept Doc: Scheduling & Orders

## Goal

Write the scheduling concept page. Cover the LLM-powered scheduling loop, mise en place, orders, stages, and how work flows from backlog to completion.

## Changes

- **`docs/concepts/scheduling.md`** — New page covering:
  - The scheduling loop (brief → schedule → orders → dispatch → cook → merge)
  - Mise en place (`mise.json`) — what it contains, why it exists
  - Orders and stages — the work unit model
  - The `orders-next.json` → `orders.json` promotion flow
  - Stage pipeline: pending → active → merging → completed/failed
  - Task types — how `schedule` frontmatter field creates them
  - Routing — provider/model/runtime per stage
  - Concurrency — parallel cooks, worktree isolation, merge serialization

## Data structures

- `orderx.Order` and `orderx.Stage` — document key fields
- `mise.Brief` — explain what the scheduler sees

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Core architectural concept, needs clear explanation |

Apply `unslop` skill.

## Verification

### Static
- Page builds without errors
- Diagrams/flow descriptions match actual code flow

### Runtime
- Concepts described match observable behavior when running `noodle start`
- File paths mentioned (`.noodle/mise.json`, `.noodle/orders.json`) exist at runtime

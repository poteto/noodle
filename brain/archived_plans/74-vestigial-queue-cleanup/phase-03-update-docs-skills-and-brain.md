Back to [[archived_plans/74-vestigial-queue-cleanup/overview]]

# Phase 3: Update docs, skills, and brain

## Goal

Update all non-code references from "queue" to "orders" terminology. These are user-facing docs and agent-facing skill instructions.

## Changes

### `README.md`

- Line 33: "schedules queue from mise data" → "schedules orders from mise data"
- Line 93: schema targets list `queue` → `orders, status`

### `INSTALL.md`

- Line 28: "`.noodle/queue.json` is the work queue" → "`.noodle/orders.json` is the work orders"

### `PHILOSOPHY.md`

- Lines 43-47: Replace `.noodle/queue.json` with `.noodle/orders.json`, "work queue" with "work orders"
- Line 53: "reads the mise and writes the queue" → "reads the mise and writes orders"

### `.agents/skills/debugging/SKILL.md`

- Line 19: `.noodle/queue.json` → `.noodle/orders.json`
- Line 41: "`.noodle/queue.json` validity" → "`.noodle/orders.json` validity"
- Line 42: "Queue stuck" → "Orders stuck", `.noodle/queue.json` → `.noodle/orders.json`

### `brain/codebase/runtime-routing-owned-by-queue.md`

Rename file to `runtime-routing-owned-by-orders.md`. Update content:
- Title: "Runtime Routing Owned By Orders"
- "queue items" → "order stages"
- "`queue.runtime`" → "stage `runtime` field"

### `internal/schemadoc/specs.go`

- Status schema description: "queue item IDs currently being cooked" → "order IDs currently being cooked"

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — judgment calls on wording, touching user-facing prose.

## Verification

```bash
go build ./...
grep -ri 'queue\.json\|"the work queue"' README.md INSTALL.md PHILOSOPHY.md .agents/skills/debugging/SKILL.md
```

Grep should return zero matches. Read through each changed file to verify the prose reads naturally with the new terminology.

# Skill Authoring

How to create and update Noodle skills — task-type skills that plug into the scheduling loop, domain skills that teach agents about the codebase, and workflow skills.

## The Pipeline

Understand this before writing any skill:

```
backlog (todos.md or adapter)
  ↓ sync
mise.json       ← backlog + task_types[] + recent_events + resources
  ↓ schedule skill reads mise, writes orders
orders-next.json ← orders with staged pipelines
  ↓ loop promotes atomically
orders.json     ← live orders dispatched to cook sessions
  ↓ skill loaded into session
agent runs      ← SKILL.md body is the agent's instructions
```

The `schedule` skill is the only writer of `orders-next.json`. Skills influence scheduling through their `schedule` frontmatter field — the scheduler reads these from `task_types` in mise.

## Skill Anatomy

```
skill-name/
├── SKILL.md          ← required: frontmatter + instructions
├── references/       ← optional: docs loaded on demand
├── scripts/          ← optional: deterministic executable code
└── assets/           ← optional: files used in output, not loaded into context
```

**SKILL.md** has two parts:
1. **Frontmatter** (YAML) — `name`, `description`, and optionally `schedule`. Only frontmatter is always in context; the body loads after triggering.
2. **Body** (markdown) — what the agent should do when this skill runs.

### Context Budget

The context window is a shared resource. Only add what Claude doesn't already know. Prefer concise examples over verbose explanations. Keep SKILL.md under 500 lines; split into references when approaching that limit.

Three-level loading:
1. **Metadata** (~100 words) — always in context
2. **SKILL.md body** (<5k words) — loaded on trigger
3. **References/scripts** — loaded on demand by the agent

### Frontmatter

```yaml
---
name: my-skill
description: >-
  What the skill does and when to trigger it. Put ALL trigger info here —
  the body only loads after triggering.
schedule: "When and where to place this task type in order pipelines"
---
```

- `name` + `description`: Required. Description is the primary trigger mechanism.
- `schedule`: Optional. Presence makes this a **task type** discoverable by the scheduling loop.

### Two Kinds of Skills

**Task-type skills** have a `schedule` field and appear in `mise.json` as `task_types`. The scheduler places them as stages in orders. Examples: execute, quality, reflect, meditate.

**Domain/workflow skills** have no `schedule` field. They're loaded alongside task-type skills to provide context. Examples: go-best-practices, react-best-practices, noodle.

## Writing Task-Type Skills

### The schedule Field

This is the scheduler's instruction for when and how to use this task type. Answer: **when should this appear, and where in the pipeline?**

Three positioning patterns:

| Pattern | When | Schedule hint |
|---------|------|---------------|
| **Follow-up stage** | Runs after another stage in the same order | `"Follow-up stage after execute. Cross-provider review preferred."` |
| **Standalone order** | Own order, triggered by conditions | `"Standalone order when [condition]. [constraints]."` |
| **Primary stage** | The main work of an order | `"When backlog items [condition]"` |

Real examples from this project:

```yaml
# quality — follow-up
schedule: "Follow-up stage after execute. Cross-provider review preferred — if codex executed, schedule quality on claude; if claude executed, schedule on codex."

# reflect — follow-up
schedule: "Follow-up stage after quality — capture learnings from the completed execute→quality cycle."

# meditate — standalone
schedule: "Standalone single-stage order after several reflect cycles have accumulated. Expensive — don't over-schedule."

# execute — primary
schedule: "When backlog items with linked plans are ready for implementation"
```

### The Skill Body

Write instructions for what the agent should do when dispatched. The body should NOT describe when to schedule itself — that's the `schedule` field's job. Keep these concerns separate.

## Orders Schema

Run `noodle schema orders` for the canonical schema. Key structure:

```json
{
  "orders": [{
    "id": "string — backlog item ID or slug",
    "title": "brief description",
    "plan": ["linked plan paths"],
    "rationale": "why scheduled, citing a principle",
    "status": "active",
    "stages": [{
      "task_key": "execute",
      "skill": "execute",
      "provider": "codex",
      "model": "gpt-5.4",
      "runtime": "sprites",
      "status": "pending",
      "prompt": "full task prompt",
      "extra_prompt": "supplemental approach instructions (~1000 chars max)"
    }]
  }]
}
```

### Stage Composition

**Sequential pipeline** — most common, stages run one at a time:
```json
[
  {"task_key": "execute", "provider": "codex", "runtime": "sprites", ...},
  {"task_key": "quality", "provider": "claude", "runtime": "process", ...},
  {"task_key": "reflect", "provider": "claude", "runtime": "process", ...}
]
```

**Prepended stage** — add before execute when preconditions apply:
```json
[
  {"task_key": "debate", "provider": "claude", ...},
  {"task_key": "execute", "provider": "codex", ...}
]
```

**Single-stage standalone** — for periodic/event-driven tasks:
```json
[{"task_key": "meditate", "provider": "claude", "runtime": "process", ...}]
```

**Parallel groups** — stages in the same `group` run concurrently:
```json
[
  {"task_key": "lint", "group": 1, ...},
  {"task_key": "test", "group": 1, ...},
  {"task_key": "deploy", "group": 2, ...}
]
```

## Full orders-next.json Examples

### Backlog item with a plan — execute → quality → reflect

```json
{
  "orders": [
    {
      "id": "49",
      "title": "implement work orders redesign",
      "plan": ["plans/49-work-orders-redesign/overview"],
      "rationale": "foundation-before-feature: core infra needed by all other work",
      "status": "active",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "codex",
          "model": "gpt-5.4",
          "runtime": "sprites",
          "status": "pending",
          "extra_prompt": "Phase 2 of the work orders redesign. Read brain/plans/49-work-orders-redesign/phase-2.md for the full brief. Phase 1 (schema types) is complete."
        },
        {
          "task_key": "quality",
          "skill": "quality",
          "provider": "claude",
          "model": "claude-opus-4-6",
          "runtime": "process",
          "status": "pending"
        },
        {
          "task_key": "reflect",
          "skill": "reflect",
          "provider": "claude",
          "model": "claude-opus-4-6",
          "runtime": "process",
          "status": "pending"
        }
      ]
    }
  ]
}
```

### Simple backlog item without a plan

```json
{
  "orders": [
    {
      "id": "73",
      "title": "fix login timeout on slow connections",
      "rationale": "high-priority bug, straightforward fix",
      "status": "active",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "codex",
          "model": "gpt-5.4",
          "runtime": "process",
          "prompt": "Fix the login timeout bug. The HTTP client in auth/client.go uses a 5s timeout which is too short for users on slow connections. Increase to 30s and add a configurable timeout option.",
          "status": "pending"
        },
        {
          "task_key": "quality",
          "skill": "quality",
          "provider": "claude",
          "model": "claude-opus-4-6",
          "runtime": "process",
          "status": "pending"
        }
      ]
    }
  ]
}
```

### Complex item that needs planning first

```json
{
  "orders": [
    {
      "id": "81",
      "title": "add real-time collaboration",
      "rationale": "complex feature, needs plan before implementation",
      "status": "active",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "claude",
          "model": "claude-opus-4-6",
          "runtime": "process",
          "status": "pending",
          "extra_prompt": "This item needs a plan before implementation. Use /plan to break it down into phases, then execute the first phase."
        }
      ]
    }
  ]
}
```

### Shared infrastructure order (not tied to a backlog item)

```json
{
  "orders": [
    {
      "id": "infra-event-system",
      "title": "shared event types used by subagent-tracking and diffs-integration",
      "rationale": "foundation-before-feature: both plan 84 and plan 86 depend on shared event types",
      "status": "active",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "codex",
          "model": "gpt-5.4",
          "runtime": "sprites",
          "status": "pending",
          "extra_prompt": "Create shared event types in internal/event/ that both subagent-tracking and diffs-integration will consume. Keep scope narrow — only the shared interfaces, not plan-specific logic."
        },
        {
          "task_key": "quality",
          "skill": "quality",
          "provider": "claude",
          "model": "claude-opus-4-6",
          "runtime": "process",
          "status": "pending"
        }
      ]
    }
  ]
}
```

### Standalone meditate order

```json
{
  "orders": [
    {
      "id": "meditate-1",
      "title": "audit brain vault after recent reflects",
      "rationale": "3 reflects accumulated, time to consolidate",
      "status": "active",
      "stages": [
        {
          "task_key": "meditate",
          "skill": "meditate",
          "provider": "claude",
          "model": "claude-opus-4-6",
          "runtime": "process",
          "status": "pending"
        }
      ]
    }
  ]
}
```

### Parallel execution groups

```json
{
  "orders": [
    {
      "id": "92",
      "title": "implement independent API endpoints",
      "plan": ["plans/92-api-endpoints/overview"],
      "rationale": "phases 2a and 2b are independent and can run concurrently",
      "status": "active",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "codex",
          "model": "gpt-5.4",
          "runtime": "sprites",
          "status": "pending",
          "group": 0,
          "extra_prompt": "Phase 2a: implement /users endpoint. See plans/92-api-endpoints/phase-2a.md."
        },
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "codex",
          "model": "gpt-5.4",
          "runtime": "sprites",
          "status": "pending",
          "group": 0,
          "extra_prompt": "Phase 2b: implement /projects endpoint. See plans/92-api-endpoints/phase-2b.md."
        },
        {
          "task_key": "quality",
          "skill": "quality",
          "provider": "claude",
          "model": "claude-opus-4-6",
          "runtime": "process",
          "status": "pending",
          "group": 1
        }
      ]
    }
  ]
}
```

### Empty orders (nothing to schedule)

```json
{
  "orders": []
}
```

## Creating a Skill

1. `mkdir -p .agents/skills/<name>`
2. Write `SKILL.md` with frontmatter. Add `schedule:` if it's a task type.
3. Add `references/` for large docs, `scripts/` for deterministic code, `assets/` for output files.
4. The loop discovers it automatically — appears in `task_types` on next mise generation.

### Project-Specific Conventions

- **Read brain at runtime.** Skills referencing brain principles should `Read` the files so they pick up changes. Don't inline principle text.
- **Don't use plan mode for plan-only skills.** `EnterPlanMode` restricts tools to read-only. For skills that produce plans as output, write plan files directly.
- **Skills live in `.agents/skills/`**. The `.claude/skills` symlink makes them visible to Claude automatically.

## Iteration

After using the skill on real tasks:

1. Notice struggles or inefficiencies
2. Before adding text to SKILL.md, ask: can this be a script, a reference file, or a structural change? Structural fixes outlast prose.
3. Test the change on a real task

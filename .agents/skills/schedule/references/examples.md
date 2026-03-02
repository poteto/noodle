# Order Examples

## Multi-stage order

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
        {"task_key": "execute", "skill": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "process", "status": "pending"},
        {"task_key": "quality", "skill": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"},
        {"task_key": "reflect", "skill": "reflect", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

## Plan-first order (no plan yet)

No `task_key` on the planning stage — avoids loading the execute skill, which would conflict with the plan skill's "stop after planning" instruction. Use `prompt` to describe the planning task directly.

```json
{
  "orders": [
    {
      "id": "97",
      "title": "adapter schema validator — plan",
      "rationale": "complex cross-cutting item without a plan; needs plan-first then adversarial review",
      "status": "active",
      "stages": [
        {"provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending",
         "prompt": "Plan backlog item #97: adapter schema validator. Use /plan to break it down into phases. Do not implement — planning only."},
        {"task_key": "adversarial-review", "skill": "adversarial-review", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

## Order with debate stage

```json
{
  "orders": [
    {
      "id": "52",
      "title": "design cache invalidation strategy",
      "rationale": "unresolved design question needs structured debate before implementation",
      "status": "active",
      "stages": [
        {"task_key": "debate", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"},
        {"task_key": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "process", "status": "pending"},
        {"task_key": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

## Shared infrastructure order

```json
{
  "orders": [
    {
      "id": "infra-event-system",
      "title": "shared event types used by subagent-tracking and diffs-integration",
      "rationale": "foundation-before-feature: both plan 84 and plan 86 depend on shared event types",
      "status": "active",
      "stages": [
        {"task_key": "execute", "skill": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "process", "status": "pending",
         "extra_prompt": "Create shared event types in internal/event/ that both subagent-tracking and diffs-integration will consume. Keep scope narrow — only the shared interfaces, not plan-specific logic."},
        {"task_key": "quality", "skill": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

## Single-stage order

```json
{
  "orders": [
    {
      "id": "meditate-1",
      "title": "audit brain vault after recent reflects",
      "rationale": "3 reflects accumulated, time to consolidate",
      "status": "active",
      "stages": [
        {"task_key": "meditate", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

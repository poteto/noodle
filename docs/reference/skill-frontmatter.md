# Skill Frontmatter

Skills are defined in SKILL.md files. The YAML frontmatter block at the top declares metadata used for resolution, routing, and scheduling.

```yaml
---
name: my-skill
description: What this skill does.
---
```

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Identifier used to invoke the skill |
| `description` | string | yes | Short summary shown in listings and the mise |
| `model` | string | no | Override the default model for this skill |
| `schedule` | string | no | Natural-language trigger that makes this skill schedulable |

## Schedulable skills

A skill with a `schedule` field becomes **schedulable**. The scheduling agent reads these descriptions and decides when to create orders for them. The schedule string is prose, not a cron expression. The scheduler interprets it in context.

```go
func (f Frontmatter) IsTaskType() bool { return f.Schedule != "" }
```

## Examples

### Scheduler skill

```yaml
---
name: schedule
description: Orders scheduler. Reads .noodle/mise.json, writes .noodle/orders-next.json.
schedule: "When orders are empty, after backlog changes, or when session history suggests re-evaluation"
---
```

### Quality gate

```yaml
---
name: quality
description: Post-cook quality gate. Reviews completed cook work for correctness, scope discipline, and principle compliance.
schedule: "After each cook session completes"
---
```

### Adversarial review with model override

```yaml
---
name: adversarial-review
description: Adversarial code review using the opposite model. Spawns 1-3 reviewers on the opposing model.
model: claude-opus-4-6
schedule: "After cook sessions that produce large diffs (200+ lines) or implement plan phases"
---
```

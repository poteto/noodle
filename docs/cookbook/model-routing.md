# Model Routing

Noodle uses a single project-wide routing default from `.noodle.toml`:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"
```

Every stage inherits these defaults unless that stage explicitly sets `provider` and `model` in `orders.json`.

## Skill-level override

If a skill should always run on a specific model, set it directly in skill frontmatter:

```yaml
---
name: deploy
description: Deploy the project after tests pass.
model: claude-opus-4-6
schedule: "After test stages pass"
---
```

Use defaults for most work, and use explicit stage/skill overrides only when a task has different cost or capability needs.

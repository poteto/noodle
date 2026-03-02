# Model Routing

Not every task needs the same model. Formatting a file is a waste of Opus. Designing an architecture is a bad fit for a small, fast model. Noodle's routing system lets you match models to tasks using named tags.

## Defaults and tags

The routing config has two layers: a default and named tag overrides.

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.mechanical]
provider = "codex"
model = "gpt-5.3-codex"

[routing.tags.review]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.fast]
provider = "claude"
model = "claude-sonnet-4-6"
```

Every stage in an order uses the default model unless it specifies a tag. When the scheduler sets `"tag": "mechanical"` on a stage, that stage runs on Codex instead of Claude.

## How tags get applied

The scheduling agent decides which tag to use per stage when it creates orders. It reads the tag definitions from the config and picks the one that fits the work:

```json
{
  "orders": [
    {
      "id": "todo-5",
      "title": "Refactor utils to use new date library",
      "stages": [
        { "skill": "execute", "status": "pending", "tag": "mechanical" },
        { "skill": "test", "status": "pending", "tag": "fast" },
        { "skill": "execute", "status": "pending", "tag": "review",
          "prompt": "Review the refactored code for correctness" }
      ]
    }
  ]
}
```

This order uses three different models for three stages of the same task. The mechanical work (find-and-replace across files) goes to Codex. Testing runs on Sonnet because speed matters more than reasoning depth. The final review stage runs on Opus because catching subtle bugs requires stronger reasoning.

## Mixing providers

Tags aren't limited to one provider. You can mix Claude, Codex, and anything else Noodle supports:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.mechanical]
provider = "codex"
model = "gpt-5.3-codex"

[routing.tags.draft]
provider = "claude"
model = "claude-sonnet-4-6"
```

The scheduling agent picks the right tool for the job. Codex is good at bulk code changes that follow a pattern. Sonnet is fast for first drafts and boilerplate. Opus is worth the cost for architecture decisions and code review.

## Skill-level overrides

Tags apply at the order stage level, but you can also set a model directly on a skill in its frontmatter:

```yaml
---
name: deploy
description: Deploy the project after tests pass.
model: claude-opus-4-6
schedule: "After test stages pass"
---
```

A skill-level `model` field overrides the routing default for that skill regardless of tags. Use this for skills where you always want a specific model. Deploy is a good example, since you never want to cut corners on deployment.

## When to use what

**Use the default** for your most common work. Set it to whatever model you'd choose if you could only use one.

**Use tags** when the scheduler should pick per-stage. Tags give the scheduling agent a vocabulary for describing cost/capability tradeoffs. Name them after the kind of work, not the model: `mechanical` is better than `codex`, because the tag name survives when you swap models.

**Use skill-level overrides** for skills that should always use a specific model. These override both the default and any tag the scheduler might set.

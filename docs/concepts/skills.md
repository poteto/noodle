# Skills

Skills are all you need to extend Noodle. Every behavior an agent can perform is a skill: scheduling work, implementing tasks, reviewing code, reflecting on mistakes.

For why this matters, see [Vision](/vision).

## The `schedule:` field

The `schedule:` field in the frontmatter is what makes a skill run autonomously. Noodle discovers every skill with a `schedule:` field and feeds those descriptions to the scheduler agent as context. The scheduler reads them alongside the current project state and decides what to dispatch.

```yaml
---
name: schedule
description: >
  Reads .noodle/mise.json, writes .noodle/orders-next.json.
  Schedules work orders based on backlog state and session history.
schedule: >
  When orders are empty, after backlog changes,
  or when session history suggests re-evaluation
---
```

Without `schedule:`, a skill is general purpose. Agents invoke it directly when they need it.

## Scheduled vs general skills

The `schedule:` field is the dividing line.

A skill **without** `schedule:` is a **general skill**. Agents invoke it directly when they need it. For example tthis `commit` skill is a general skill: agents call it when they need to commit code, but nothing triggers it automatically.

```yaml
---
name: commit
description: >
  Create conventional commit messages following best conventions.
---
```

A skill **with** `schedule:` runs autonomously in the noodle loop. The scheduling agent reads the `schedule:` value, matches it against current conditions, and creates orders when the trigger fits. Some examples:

```yaml
# execute: runs when there's work to do
schedule: >
  When backlog items with linked plans
  are ready for implementation

# reflect: captures learnings after a session
schedule: >
  After an agent session completes

# quality: reviews output before merge
schedule: >
  After each agent session completes
```

The scheduler does not parse these strings mechanically. It reads them as prose and uses judgment. You can write conditions like "When the backlog has high-priority items", "After a failed session", or "On Mondays" and the scheduling agent figures out the rest.

## Skill Discovery

Noodle discovers skills by scanning configured search paths. The default is `.agents/skills/`. Configure additional paths in `.noodle.toml`:

```toml
[skills]
paths = [".agents/skills", ".claude/skills", "~/shared-skills"]
```

A directory counts as a skill only if it contains a `SKILL.md` file.

### Resolution Order

When resolving a skill by name, Noodle walks the `paths` list top to bottom:

1. Check if `<path>/<name>/` exists and contains a `SKILL.md`
2. If yes, use this skill. Stop.
3. Otherwise, try the next path.

First match wins. If `.agents/skills/deploy/` and `.claude/skills/deploy/` both exist, the one in `.agents/skills/` takes precedence because it appears first in the default paths.

This means you can override any built-in skill. Put your replacement in a higher-priority path, and the resolver picks it up instead.

## Composition

Skills compose naturally. You can call skills from other skills. For example, if you want your `review` skill to use the `debugging` skill, you can ask your agent to include it. `Skill(name)` seems to work best as a way to reliably get skills invoked.

If you want skills to be scheduled in a certain order, you specify that in the `schedule` skill. For example, you can say that `review` skill should always follow `execute`. See [Scheduling](/concepts/scheduling) for more details and tips.

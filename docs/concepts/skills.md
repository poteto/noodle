# Skills

Skills are Noodle's single extension point. Every behavior an agent can perform — scheduling work, executing tasks, reviewing code, reflecting on mistakes — is a skill. There are no plugins, no hooks API, no configuration DSL. Just a directory with a markdown file.

For why this matters, see [Why Noodle](/why-noodle).

## Anatomy of a Skill

A skill is a directory containing:

- **`SKILL.md`** (required) — YAML frontmatter for metadata, markdown body for the prompt
- **`references/`** (optional) — supporting files the agent can read during execution

The markdown body is the prompt. Write it as instructions for an agent: what to read, what to do, what to produce. Here is the frontmatter from the `schedule` skill that ships with Noodle:

```yaml
---
name: schedule
description: Orders scheduler. Reads .noodle/mise.json, writes .noodle/orders-next.json.
schedule: "When orders are empty, after backlog changes, or when session history suggests re-evaluation"
---
```

And the body starts with concrete instructions:

```markdown
# Schedule

Read `.noodle/mise.json`, write `.noodle/orders-next.json`.
The loop atomically promotes `orders-next.json` into `orders.json` — never write `orders.json` directly.
```

## Frontmatter

The YAML frontmatter block declares metadata. Four fields, matching the `Frontmatter` struct in `skill/frontmatter.go`:

| Field         | Required | Description                                           |
| ------------- | -------- | ----------------------------------------------------- |
| `name`        | yes      | Identifier used to invoke the skill                   |
| `description` | yes      | Short summary shown in listings and the mise          |
| `model`       | no       | Override the default model for this skill              |
| `schedule`    | no       | Natural-language trigger — makes this a task type      |

## Task-Type Skills vs General Skills

The `schedule` field is the dividing line.

A skill **without** `schedule` is a **general skill**. It gets invoked directly — by a user, by another agent, or by name in an order stage. The `commit` skill is a general skill: agents call it when they need to commit code, but nothing triggers it automatically.

```yaml
---
name: commit
description: Create conventional commit messages following best conventions.
---
```

A skill **with** `schedule` is a **task-type skill**. The scheduling agent reads the `schedule` value, matches it against current conditions, and creates orders when the trigger fits. Five task-type skills ship with Noodle:

```yaml
# execute — runs when there's work to do
schedule: "When backlog items with linked plans are ready for implementation"

# reflect — captures learnings after a session
schedule: "After a cook session completes"

# quality — reviews cook output before merge
schedule: "After each cook session completes"

# adversarial-review — cross-model code review for large changes
schedule: "After cook sessions that produce large diffs (200+ lines) or implement plan phases"

# schedule — the scheduler itself, also a task type
schedule: "When orders are empty, after backlog changes, or when session history suggests re-evaluation"
```

The scheduler does not parse these strings mechanically. It reads them as prose and uses judgment. You can write conditions like "When the backlog has high-priority items" or "After a failed execute" and the scheduling agent figures out the rest.

## Skill Discovery

Noodle discovers skills by scanning configured search paths. The default is `.agents/skills/`. Configure additional paths in `.noodle.toml`:

```toml
[skills]
paths = [".agents/skills", ".claude/skills", "~/shared-skills"]
```

A directory counts as a skill only if it contains a `SKILL.md` file.

## Resolution Order

When resolving a skill by name, Noodle walks the `paths` list top to bottom:

1. Check if `<path>/<name>/` exists and contains a `SKILL.md`
2. If yes, use this skill. Stop.
3. Otherwise, try the next path.

First match wins. If `.agents/skills/deploy/` and `.claude/skills/deploy/` both exist, the one in `.agents/skills/` takes precedence because it appears first in the default paths.

This means you can override any built-in skill. Put your replacement in a higher-priority path, and the resolver picks it up instead.

## Reference Files

A skill can include a `references/` subdirectory with supporting files — schema definitions, example configs, code snippets, anything the agent needs while executing. The `quality` skill, for example, includes a `stage-message-schema.md` reference so the agent knows the exact event format to emit.

```
.agents/skills/
  quality/
    SKILL.md
    references/
      stage-message-schema.md
```

## Creating a Skill

A minimal skill:

```
.agents/skills/
  greet/
    SKILL.md
```

```yaml
---
name: greet
description: Say hello to the user
---

Print a greeting message. Be brief.
```

To make it a task type, add `schedule`:

```yaml
---
name: greet
description: Say hello to the user
schedule: "At the start of every new session"
---

Print a greeting message. Be brief.
```

The scheduling agent now sees `greet` as an available task type and can create orders for it.

## Composition

Skills compose through the file system. The `schedule` skill reads `.noodle/mise.json` and writes orders. The `execute` skill picks up a task and does the work. The `reflect` skill reviews what happened and writes to the brain. Each skill reads from and writes to disk — the contract between skills is files.

This is intentional. No skill imports another skill. No skill calls another skill's API. They coordinate through shared state on the file system, which means you can swap out any piece without touching the others.

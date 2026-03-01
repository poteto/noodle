# Skills

Skills are Noodle's single extension point. A skill is a directory containing a `SKILL.md` file — markdown that teaches an agent how to do something. There are no plugins, no hooks API, no configuration DSL.

## Anatomy of a Skill

A skill directory contains:

- **`SKILL.md`** (required) — the skill definition, with optional YAML frontmatter and markdown body
- **`references/`** (optional) — supplementary files the agent can read for context

The `SKILL.md` body is the prompt. Write it as instructions for an agent: what to read, what to do, what to produce.

## Frontmatter

The YAML frontmatter block at the top of `SKILL.md` declares metadata:

```yaml
---
name: deploy
description: Deploy the project after a successful run on main
model: claude-sonnet-4-6
schedule: "After a successful execute completes on main branch"
---
```

| Field         | Required | Description                                           |
| ------------- | -------- | ----------------------------------------------------- |
| `name`        | yes      | Identifier used to invoke the skill                   |
| `description` | yes      | Short summary shown in listings and the mise          |
| `model`       | no       | Override the default model for this skill              |
| `schedule`    | no       | Natural-language trigger that makes this a task type   |

## Task-Type Skills vs General Skills

A skill without a `schedule` field is a **general skill**. It gets invoked directly — by a user, by another agent, or by name in an order.

A skill with a `schedule` field is a **task-type skill**. The `schedule` value is a natural-language description of when this skill should run. The scheduling agent reads these descriptions and decides when to create orders for them.

```yaml
---
name: reflect
description: Review the session and write learnings to the brain
schedule: "After every completed session"
---
```

The scheduler does not parse the `schedule` string mechanically. It reads it as prose and uses judgment to decide when the trigger conditions are met. This means you can write conditions like "When the backlog has high-priority items" or "After a failed execute" and the scheduling agent figures out the rest.

## Skill Discovery

Noodle discovers skills by scanning configured search paths. Each path is a directory containing skill subdirectories.

The default search path is `.agents/skills`. Configure additional paths in `.noodle.toml`:

```toml
[skills]
paths = [".agents/skills", ".claude/skills", "~/shared-skills"]
```

The resolver scans each path in order. For a given skill name, the first match wins. If `.agents/skills/deploy/` and `.claude/skills/deploy/` both exist, the one in `.agents/skills/` takes precedence.

A directory counts as a skill only if it contains a `SKILL.md` file.

## Resolution Order

When resolving a skill by name, the resolver walks the `paths` list top to bottom:

1. Check if `<path>/<name>/` exists and is a directory
2. Check if it contains a `SKILL.md` file
3. If both conditions pass, use this skill. Stop searching.
4. Otherwise, try the next path.

If no path contains a matching skill, resolution fails.

## Reference Files

A skill can include a `references/` subdirectory with supporting files. These might be schema definitions, example configs, code snippets, or anything else the agent needs to reference while executing the skill.

The agent receives the skill body as its prompt and can read reference files as needed.

## Creating a Skill

A minimal skill has one file:

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

To make it schedulable, add a `schedule` field:

```yaml
---
name: greet
description: Say hello to the user
schedule: "At the start of every new session"
---

Print a greeting message. Be brief.
```

The scheduling agent now sees `greet` as an available task type and can create orders for it when it judges the trigger condition is met.

## Composition

Skills compose through files. A scheduling skill reads the mise and writes orders. An execution skill picks up a task and does the work. A reflection skill reviews what happened and writes to the brain. Each skill reads from and writes to the file system — the contract between skills is files on disk.

You can replace any built-in skill with your own version. Put your replacement in a higher-priority search path, and the resolver picks it up instead of the default.

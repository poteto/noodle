# Self-Learning

Install [brainmaxxing](https://github.com/poteto/brainmaxxing) to add this optional upgrade to Noodle.

This skill pack adds a brain, a directory of markdown files committed to your repo at `brain/`. It stores project knowledge: principles, patterns, past mistakes, codebase quirks, architectural decisions. With this skill pack installed, agents read the brain before they start working and write to it when they learn something.

**Noodle works without this**. The noodle loop does not depend on it. This is entirely additive, and you can skip this is if you prefer.

The brain becomes useful when you want agents to share context across sessions: avoiding repeated mistakes, following project-specific conventions, building on past decisions.

## Installation

Tell your agent:

```md
Install brainmaxxing from https://github.com/poteto/brainmaxxing into this project.
```

This copies a `brain/` starter vault, the `reflect`/`meditate`/`ruminate` skills, and hooks into your project.

After installing, add `schedule:` fields to the brainmaxxing skills so the noodle loop runs them automatically. For example, tell your agent:

```md
Add `schedule:` fields to the reflect, meditate, and ruminate skills.
Reflect should run after every agent session. Meditate should run
periodically after several reflects accumulate. Ruminate should not
be scheduled but run ad-hoc to mine past conversations.
```

## Structure

The brain is an [Obsidian](https://obsidian.md)-compatible vault. One topic per file, organized by directory, cross-referenced with `[[wikilinks]]`.

A typical brain looks like:

```
brain/
  index.md           # top-level index with links to sections
  principles.md      # project principles
  todos.md           # tracked work items
  codebase/
    index.md
    api-patterns.md
    test-conventions.md
  plans/
    index.md
    migration-v2.md
```

Installing [brainmaxxing](https://github.com/poteto/brainmaxxing) creates a starter vault with `index.md`, `principles.md`, and `todos.md`.

## How Agents Use It

Agents treat the brain as shared memory:

- Before starting work, an agent reads relevant brain files for context
- During work, if an agent encounters something notable, it writes a brain note immediately
- After work, the reflect skill reviews the session and encodes learnings

The brain accumulates your project's actual working knowledge. It replaces the knowledge that usually lives in someone's head and gets lost when they leave.

## The Self-Learning Loop

Brainmaxxing adds three skills that form a learning loop. Reflect runs after every session and captures what happened. Meditate runs periodically and distills accumulated reflections into principles. Ruminate mines past conversations for knowledge that was never written down. See the [Self-Learning cookbook](/cookbook/self-learning) for details.

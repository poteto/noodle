# Brain

The brain is a directory of markdown files committed to your repo at `brain/`. It stores everything your project has learned — principles, patterns, past mistakes, codebase quirks, architectural decisions. Agents read the brain before they start working and write to it after they learn something.

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

Noodle creates a minimal brain on first run with `index.md`, `principles.md`, and `todos.md`.

## Why Files

The brain is in git because it should evolve with the project. Architecture changes, the brain updates. A principle proves wrong, it gets deleted. When you roll back code, you roll back knowledge too.

Because the brain is markdown files, agents interact with it the same way they interact with code — read a file, write a file. No special API, no database queries.

## The Self-Learning Loop

Three skills form the learning loop:

**Reflect** runs after significant sessions. The agent reviews what happened — what worked, what broke, what was surprising — and writes notes to the brain. Maybe a test pattern keeps failing, or an API has a quirk that wastes time. The next agent reads that note and avoids the same trap.

**Meditate** runs periodically. It audits the brain for stale content, discovers cross-cutting principles from individual notes, and prunes what is no longer true. Over time, isolated observations condense into principles that shape future decisions. Meditate also encodes these principles back into your skills, so every future agent session is grounded by them.

**Ruminate** mines past conversations for knowledge that was never captured. Corrections, workarounds, patterns that someone figured out but never wrote down. It cross-references with the existing brain to fill gaps.

Nobody curates this manually. Each session leaves the brain a little better than it found it.

## How Agents Use It

Agents treat the brain as shared memory:

- Before starting work, an agent reads relevant brain files for context about the project
- During work, if an agent encounters something notable, it can write a brain note immediately
- After work, the reflect skill reviews the session and encodes learnings

The brain accumulates your project's actual working knowledge — lessons that agents actively learn from and contribute to. It replaces the knowledge that usually lives in someone's head and gets lost when they leave.

## Wikilinks

Brain files use `[[wikilinks]]` to reference each other. An index file might link to its children:

```markdown
# Codebase

- [[api-patterns]]
- [[test-conventions]]
- [[error-handling]]
```

This keeps the vault navigable in Obsidian and lets agents follow links to find related context.

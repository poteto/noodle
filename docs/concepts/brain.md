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

## How Agents Use It

Agents treat the brain as shared memory:

- Before starting work, an agent reads relevant brain files for context
- During work, if an agent encounters something notable, it writes a brain note immediately
- After work, the reflect skill reviews the session and encodes learnings

The brain accumulates your project's actual working knowledge. It replaces the knowledge that usually lives in someone's head and gets lost when they leave.

## Self-Learning

The brain powers Noodle's self-learning loop through three skills — reflect, meditate, and ruminate. See [Vision](/vision) for how they work together.

## Why Files

The brain lives in git because it should evolve with the project. Architecture changes, the brain updates. A principle proves wrong, it gets deleted. When you roll back code, you roll back knowledge too.

Because the brain is markdown, agents interact with it the same way they interact with code. No special API, no database queries.

## Wikilinks

Brain files use `[[wikilinks]]` to reference each other. An index file might link to its children:

```markdown
# Codebase

- [[api-patterns]]
- [[test-conventions]]
- [[error-handling]]
```

This keeps the vault navigable in Obsidian and lets agents follow links to find related context.

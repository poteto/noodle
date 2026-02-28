---
name: todo
description: Add, complete, or view items in the brain/todos.md backlog. Use when the user says "todo", "add a todo", "mark done", "what's on the backlog", or wants to manage their task list.
---

# Todo

Manage the project backlog at `brain/todos.md`. Edit the file directly with Edit/Write tools.

## File Format

```markdown
# Todos

<!-- next-id: 26 -->

## Section Name

1. [x] Completed item description
2. [ ] Open item description [[plans/02-some-plan/overview]]
```

- Items are numbered markdown checkboxes: `N. [ ] description` or `N. [x] description`
- Sections are `## Heading` groups
- `<!-- next-id: N -->` tracks the next available ID

## Operations

### Add an item

1. Read `<!-- next-id: N -->` to get the next ID
2. Append `N. [ ] description` to the target section
3. Increment the next-id marker: `<!-- next-id: N+1 -->`

### Mark done

Change `[ ]` to `[x]` and update the description with a reason:

```
2. [x] Old description — superseded by #75 channel UI redesign
```

Then move the done item from `brain/todos.md` to `brain/archive/todos.md`. Remove empty sections left behind.

### Edit description

Replace the description text. Preserve any `[[wikilinks]]` unless intentionally removing them.

## Rules

- IDs are permanent. Never renumber or reuse them.
- Always increment `<!-- next-id -->` when adding items.
- Keep items in their section — don't move between sections without being asked.

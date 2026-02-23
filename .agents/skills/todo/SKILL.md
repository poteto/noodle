---
name: todo
description: Add, complete, or view items in the brain/todos.md backlog. Use when the user says "todo", "add a todo", "mark done", "what's on the backlog", or wants to manage their task list.
---

# Todo

Manage the project backlog at `brain/todos.md` via the `noodle todo` CLI.

## Commands

All mutations go through the CLI to ensure consistent ID allocation and format validation.

### Add an item

```bash
noodle todo add --section "Section Name" "Item description"
```

Omit `--section` to append to the last section. The CLI auto-assigns the next available ID and prints it.

### Mark an item done

```bash
noodle todo done ID
```

Wraps the item in `~~strikethrough~~`. The item remains in the file for history.

### Move an item to another section

```bash
noodle todo move --section "Target Section" ID
```

### Edit an item's description

```bash
noodle todo edit ID "New description"
```

Preserves wikilinks if included in the new description.

### List all items

```bash
noodle todo list
```

Shows active items grouped by section with their IDs.

## Rules

- Never edit `brain/todos.md` directly with Edit/Write tools. Always use `noodle todo` commands.
- IDs are permanent and auto-assigned. Never renumber or reuse them.
- The `<!-- next-id: N -->` marker is managed by the CLI. Do not modify it manually.

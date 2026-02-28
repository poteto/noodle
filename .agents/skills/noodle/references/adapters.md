# Adapter Reference

Adapters bridge your backlog or plan system to Noodle. Each adapter has a skill (teaches agents the semantics) and scripts (deterministic commands for CRUD actions). Adapters are optional -- if omitted, the mise contains only internal state.

## Adapter config

```toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
add = ".noodle/adapters/backlog-add"
done = ".noodle/adapters/backlog-done"
edit = ".noodle/adapters/backlog-edit"
```

Scripts can be any executable -- shell scripts, binaries, or inline commands like `gh issue close`. Noodle calls them mechanically; the skill teaches agents when and why to use them.

## Writing adapter scripts

Each adapter action maps to a script path in config. Scripts receive arguments via environment variables and must produce NDJSON output for sync actions.

1. **Sync** -- reads all items from your system, writes NDJSON to stdout. Each line is a `BacklogItem` (with optional `plan` field pointing to a plan file).
2. **Add** -- creates a new item. Receives `NOODLE_TITLE` and `NOODLE_BODY` env vars.
3. **Done** -- marks an item complete. Receives `NOODLE_ID`.
4. **Edit** -- updates an item. Receives `NOODLE_ID`, `NOODLE_FIELD`, `NOODLE_VALUE`.

## Markdown backlog (default)

The default adapter reads `brain/todos.md` -- a markdown file with numbered items. Scripts live at `.noodle/adapters/backlog-*`.

## GitHub Issues

```toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = "gh issue list --json number,title,body,labels,state"
add = "gh issue create"
done = "gh issue close"
edit = "gh issue edit"
```

## Linear

Use the Linear CLI or API. The adapter pattern is the same -- write scripts that call the Linear API and output NDJSON.

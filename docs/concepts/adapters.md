# Adapters

Adapters bridge external systems into Noodle's backlog. If your team tracks work in GitHub Issues, Linear, Jira, or somewhere else, an adapter syncs those items so Noodle can schedule and execute them.

The default backlog is a markdown file (`brain/todos.md`). You don't need an adapter if that's all you use. Adapters are for when your tickets already live somewhere else and you don't want to duplicate them.

## How they work

An adapter is a set of commands configured in `.noodle.toml`. Each command handles one operation:

- **sync**: pulls items from the external source into Noodle's backlog format
- **done**: marks an item complete in the external system
- **add** _(planned)_: creates a new item in the external system
- **edit** _(planned)_: updates an existing item

```toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = "my-adapters/backlog-sync"
add = "my-adapters/backlog-add"
done = "my-adapters/backlog-done"
edit = "my-adapters/backlog-edit"
```

Commands are executed relative to the project root via `sh -c`. They can be anything that runs in a shell: a POSIX shell script, a Python script, a Node script, a compiled Go binary, a `curl` one-liner. As long as `sh -c <your-command>` can invoke it, it works.

## When the scheduler runs

The scheduler doesn't care where backlog items come from. It reads the backlog, and if an adapter is configured, the sync script runs first to pull in fresh items. From the scheduler's perspective, the backlog is the backlog regardless of source.

This means you can switch from `todos.md` to GitHub Issues (or vice versa) without changing your skills. The adapter handles the translation.

## Schema

Each adapter command has a specific contract for input and output.

### `sync`

No input. Prints newline-delimited JSON (NDJSON) to stdout, one backlog item per line.

```json
{
  "id": "1",
  "title": "Fix login bug",
  "status": "open",
  "tags": ["bug", "auth"],
  "priority": 1
}
{
  "id": "2",
  "title": "Update docs",
  "status": "done",
  "section": "Documentation"
}
{
  "id": "3",
  "title": "Refactor API",
  "estimate": "medium"
}
```

Only `id` and `title` are required. Everything else — any field, any type — is passed through to mise.json for the scheduler to see.

| Field    | Type   | Required | Notes                              |
| -------- | ------ | -------- | ---------------------------------- |
| `id`     | string | yes      | unique identifier                  |
| `title`  | string | yes      | item title                         |
| `status` | string | no       | used by Noodle to filter done items |
| `plan`   | string | no       | used by Noodle to read plan files  |

`status` and `plan` are the only optional fields Noodle interprets. All other fields (e.g. `tags`, `section`, `estimate`, `priority`, or anything your system tracks) are passed through as-is. The scheduler sees them in mise.json and can use them for context, but Noodle itself doesn't act on them.

### `done`

Noodle calls your `done` command when an order completes, passing the item ID as the first argument. Your command should mark that item complete in your external system. No stdin, no output expected.

```
my-adapters/backlog-done <id>
```

### `add` _(planned)_

Not yet called by the noodle loop. When implemented, your `add` command will receive JSON on stdin and should create the item in your external system, then print the new item's ID to stdout.

### `edit` _(planned)_

Not yet called by the noodle loop. When implemented, your `edit` command will receive the item ID as the first argument and JSON on stdin with the new values.

## Validation

Noodle validates each NDJSON line from the sync command individually. Invalid items are skipped — they don't block the rest of the backlog from processing.

**What gets validated:**
- Each line must be valid JSON
- `id` must be a non-empty string
- `title` must be a non-empty string

**What happens on failure:**
- Invalid items are skipped and a warning is generated for each one
- Valid items from the same sync still process normally
- If every item is invalid, the backlog is empty but no error is raised

**Where warnings surface:**
- **Web UI** — warnings appear in the dashboard warnings panel alongside config warnings
- **Backend logs** — logged as a `sync.degraded` event
- **Scheduler prompt** — the scheduler sees an `ADAPTER WARNINGS` section and may create a fix task

**Example warnings:**
```
backlog sync line 3: invalid JSON: unexpected end of JSON input
backlog sync line 5: missing required field title
backlog sync line 12: missing required field id
```

## Writing an adapter

A minimal GitHub Issues adapter might use `gh` CLI to list open issues tagged with a label, format them as backlog items, and print NDJSON to stdout. The commands are simple enough that an agent can write them for you. Point it at your issue tracker and the schema above and it'll produce the scripts.

## Configuration reference

See [`[adapters.<name>]`](/reference/configuration#adapters-name) for the full config spec.

# Adapters

Adapters bridge external systems into Noodle's backlog. If your team tracks work in GitHub Issues, Linear, Jira, or somewhere else, an adapter syncs those items so Noodle can schedule and execute them.

The default backlog is a markdown file (`brain/todos.md`). You don't need an adapter if that's all you use. Adapters are for when your tickets already live somewhere else and you don't want to duplicate them.

## How they work

An adapter is a set of shell scripts configured in `.noodle.toml`. Each script handles one operation:

- **sync**: pulls items from the external source into Noodle's backlog format
- **add**: creates a new item in the external system
- **done**: marks an item complete in the external system
- **edit**: updates an existing item

```toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
add = ".noodle/adapters/backlog-add"
done = ".noodle/adapters/backlog-done"
edit = ".noodle/adapters/backlog-edit"
```

Scripts are executed relative to the project root. Each script receives structured input on stdin and produces structured output on stdout. Noodle runs them via `sh -c`, so they don't need to be `chmod +x`. Just valid shell scripts.

## When the scheduler runs

The scheduler doesn't care where backlog items come from. It reads the backlog, and if an adapter is configured, the sync script runs first to pull in fresh items. From the scheduler's perspective, the backlog is the backlog regardless of source.

This means you can switch from `todos.md` to GitHub Issues (or vice versa) without changing your skills. The adapter handles the translation.

## Writing an adapter

An adapter is just shell scripts that speak Noodle's backlog format. A minimal GitHub Issues adapter might use `gh` CLI to list open issues tagged with a label, format them as backlog items, and print to stdout.

The scripts are simple enough that an agent can write them for you. Point it at your issue tracker and the [configuration reference](/reference/configuration#adapters-name) and it'll produce the scripts.

## Configuration reference

See [`[adapters.<name>]`](/reference/configuration#adapters-name) for the full config spec.

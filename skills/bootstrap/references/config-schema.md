# Noodle Config Schema Reference

Bootstrap writes `noodle.toml` at project root.

Use these defaults unless the project already has a valid value.

## Minimal Starter Config

```toml
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[skills]
paths = ["skills", "~/.noodle/skills"]

[agents]
claude_dir = ""
codex_dir = ""

[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
add = ".noodle/adapters/backlog-add"
done = ".noodle/adapters/backlog-done"
edit = ".noodle/adapters/backlog-edit"
```

Add plans adapter only when the user confirms they use plans:

```toml
[adapters.plans]
skill = "plans"

[adapters.plans.scripts]
sync = ".noodle/adapters/plans-sync"
create = ".noodle/adapters/plan-create"
done = ".noodle/adapters/plan-done"
phase-add = ".noodle/adapters/plan-phase-add"
```

## Full Default Values

```toml
[phases]
oops = "oops"
debugging = "debugging"

[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
add = ".noodle/adapters/backlog-add"
done = ".noodle/adapters/backlog-done"
edit = ".noodle/adapters/backlog-edit"

[adapters.plans]
skill = "plans"

[adapters.plans.scripts]
sync = ".noodle/adapters/plans-sync"
create = ".noodle/adapters/plan-create"
done = ".noodle/adapters/plan-done"
phase-add = ".noodle/adapters/plan-phase-add"

[sous-chef]
skill = "sous-chef"
run = "after-each"
model = "claude-sonnet"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[routing.tags]

[skills]
paths = ["skills", "~/.noodle/skills"]

[review]
enabled = true

[recovery]
max_retries = 3
retry_suffix_pattern = "-recover-%d"

[monitor]
stuck_threshold = "120s"
ticket_stale = "30m"
poll_interval = "5s"

[concurrency]
max_cooks = 4

[agents]
claude_dir = ""
codex_dir = ""
```

## Repair Rules

- Keep existing valid values.
- Add missing required fields.
- Remove plans adapter only when explicitly asked or in non-interactive fallback mode.
- Never write secrets.

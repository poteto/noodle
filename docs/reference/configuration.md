# Configuration

Noodle is configured through a `.noodle.toml` file at the root of your project. When no config file exists, Noodle uses sensible defaults for all values.

## Minimal config

Most projects only need routing defaults and a skills path:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
```

## Full reference

### `mode`

Controls human oversight level.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mode` | string | `"auto"` | Autonomy mode: `"auto"`, `"supervised"`, or `"manual"` |

```toml
mode = "auto"
```

---

### `[routing]`

Controls which LLM provider and model are used for skill execution. Routing has global defaults and optional per-tag overrides.

#### `[routing.defaults]`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `provider` | string | `"claude"` | LLM provider name (e.g. `"claude"`, `"codex"`) |
| `model` | string | `"claude-opus-4-6"` | Model identifier passed to the provider |

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"
```

#### `[routing.tags.<name>]`

Tag-based routing overrides. When a skill or ticket is tagged, the matching policy takes precedence over defaults.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `provider` | string | — | LLM provider for this tag |
| `model` | string | — | Model for this tag |

```toml
[routing.tags.mechanical]
provider = "codex"
model = "gpt-5.3-codex"

[routing.tags.urgent]
provider = "claude"
model = "claude-opus-4-6"
```

---

### `[skills]`

Configures where Noodle discovers skill definitions.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `paths` | string[] | `[".agents/skills"]` | Directories to scan for SKILL.md files |

```toml
[skills]
paths = [".agents/skills"]
```

---

### `[recovery]`

Controls automatic retry behavior when an agent session fails.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_retries` | int | `3` | Maximum retry attempts before marking a ticket as failed |

```toml
[recovery]
max_retries = 3
```

---

### `[monitor]`

Configures the monitoring loop that detects stuck sessions and stale tickets.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stuck_threshold` | duration | `"120s"` | Time without progress before a session is considered stuck |
| `ticket_stale` | duration | `"30m"` | Time before an unfinished ticket is flagged as stale |
| `poll_interval` | duration | `"5s"` | How often the monitor checks session health |

```toml
[monitor]
stuck_threshold = "120s"
ticket_stale = "30m"
poll_interval = "5s"
```

---

### `[concurrency]`

Controls parallel execution limits and backpressure.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_cooks` | int | `4` | Maximum number of concurrent agent sessions |
| `max_completion_overflow` | int | `1024` | Buffer size for completion events before backpressure kicks in |
| `merge_backpressure_threshold` | int | `128` | Pending merges before new completions are throttled |
| `shutdown_timeout` | duration | `"30s"` | Grace period for running sessions during shutdown |

```toml
[concurrency]
max_cooks = 4
max_completion_overflow = 1024
merge_backpressure_threshold = 128
shutdown_timeout = "30s"
```

::: warning Cost
Each agent is a full LLM session. `max_cooks = 4` means up to four concurrent API sessions, each consuming tokens independently. Start with `max_cooks = 1` or `2` while you're learning the system, then scale up once you've seen the cost per session on your workload.
:::

---

### `[agents]`

Configures paths and arguments for LLM agent binaries. Each sub-table names a provider.

#### `[agents.claude]`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | string | `""` | Path to the Claude CLI binary or directory |
| `args` | string[] | `[]` | Additional CLI arguments passed on every invocation |

```toml
[agents.claude]
path = "~/.claude"
args = ["--dangerously-skip-permissions", "--verbose"]
```

#### `[agents.codex]`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | string | `""` | Path to the Codex CLI binary or directory |
| `args` | string[] | `[]` | Additional CLI arguments passed on every invocation |

```toml
[agents.codex]
path = "~/.codex"
```

---

### `[runtime]`

Controls which runtime executes agent sessions and per-runtime settings.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default` | string | `"process"` | Default runtime kind: `"process"`, `"sprites"`, or `"cursor"` |

```toml
[runtime]
default = "process"
```

#### `[runtime.process]`

Local process runtime. Runs agent CLIs as child processes on the host machine.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_concurrent` | int | `4` | Maximum concurrent process sessions |

```toml
[runtime.process]
max_concurrent = 4
```

#### `[runtime.sprites]`

Cloud runtime using [Sprites](https://sprites.dev) sandboxed environments.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `token_env` | string | `"SPRITES_TOKEN"` | Environment variable name for the Sprites API token |
| `base_url` | string | `""` | Custom API base URL (leave empty for default) |
| `sprite_name` | string | `""` | Name prefix for spawned sprite instances |
| `git_token_env` | string | `"GITHUB_TOKEN"` | Environment variable name for the Git token used by sprites |
| `max_concurrent` | int | `50` | Maximum concurrent sprite sessions |

```toml
[runtime.sprites]
token_env = "SPRITES_TOKEN"
base_url = ""
sprite_name = "noodle-dev"
git_token_env = "GITHUB_TOKEN"
max_concurrent = 50
```

#### `[runtime.cursor]`

Cloud runtime using [Cursor](https://cursor.com) background agents.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `api_key_env` | string | `"CURSOR_API_KEY"` | Environment variable name for the Cursor API key |
| `base_url` | string | `""` | Custom API base URL (leave empty for default) |
| `repository` | string | `""` | Repository identifier for Cursor sessions |
| `max_concurrent` | int | `10` | Maximum concurrent Cursor sessions |

```toml
[runtime.cursor]
api_key_env = "CURSOR_API_KEY"
base_url = ""
repository = "owner/repo"
max_concurrent = 10
```

---

### `[server]`

Controls the web UI server.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | `3000` | Port for the web UI server |
| `enabled` | bool | auto | Whether to start the server. When omitted, the server starts automatically in interactive terminals |

```toml
[server]
port = 3000
enabled = true
```

---

### `[adapters.<name>]`

Adapters bridge external systems (issue trackers, project boards) into Noodle's backlog. Each adapter is a named table mapping a skill to a set of shell scripts.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `skill` | string | — | Skill name this adapter extends |
| `scripts` | map | — | Named shell scripts the adapter invokes |

The default config includes a `backlog` adapter:

```toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
add = ".noodle/adapters/backlog-add"
done = ".noodle/adapters/backlog-done"
edit = ".noodle/adapters/backlog-edit"
```

Scripts are executed relative to the project root. Each script receives structured input on stdin and must produce structured output on stdout. See the [backlog skill docs](/concepts/skills) for the expected interface.

## Common configurations

### Local development with Claude

Minimal setup for a single developer using Claude as the LLM provider:

```toml
mode = "auto"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]

[agents.claude]
path = "~/.claude"

[concurrency]
max_cooks = 2
```

### Cloud execution with Sprites

Scale out with cloud sandboxes while keeping a local fallback:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[runtime]
default = "sprites"

[runtime.sprites]
sprite_name = "my-project"
max_concurrent = 20

[runtime.process]
max_concurrent = 2
```

### Mixed routing with tag overrides

Route mechanical tasks to a cheaper model, keep complex work on a capable one:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.mechanical]
provider = "codex"
model = "gpt-5.3-codex"

[routing.tags.review]
provider = "claude"
model = "claude-opus-4-6"
```

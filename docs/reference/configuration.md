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

Controls which Agent provider and model are used for skill execution.

#### `[routing.defaults]`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `provider` | string | `"claude"` | Agent provider name (e.g. `"claude"`, `"codex"`) |
| `model` | string | `"claude-opus-4-6"` | Model identifier passed to the provider |

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"
```

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

### `[concurrency]`

Controls parallel execution limits.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_concurrency` | int | `4` | Maximum number of concurrent agent sessions |

```toml
[concurrency]
max_concurrency = 4
```

::: warning Cost
Each agent consumes tokens independently. `max_concurrency = 4` means up to four concurrent API sessions. Start with `max_concurrency = 1` or `2` while you're learning the system, then scale up once you've seen the cost per session on your workload.
:::

---

### `[agents]`

Configures paths and arguments for agent CLI binaries. Each sub-table names a provider.

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
args = ["--full-auto"]
```

---

### `[runtime]`

Controls which runtime executes agent sessions and per-runtime settings.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default` | string | `"process"` | Default runtime kind: `"process"` or `"sprites"` |

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

---

### `[server]`

Controls the web UI server.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | `3000` | Port for the web UI server |
| `enabled` | bool | `true` | Whether to start the server. When omitted, the server starts automatically |

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
sync = "my-adapters/backlog-sync"
done = "my-adapters/backlog-done"
```

Scripts are executed relative to the project root. Each script receives structured input on stdin and must produce structured output on stdout. See the [backlog skill docs](/concepts/skills) for the expected interface.

## Common configurations

### Local development with Claude

Minimal setup for a single developer using Claude as the Agent provider:

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
max_concurrency = 2
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

### Defaults-only routing

Set a single project-wide default provider and model:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"
```

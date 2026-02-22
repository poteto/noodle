# Noodle Config

## File Names and Locations

Noodle loads config in this precedence order:

1. Global: `~/.noodle/config.toml`
2. Project: `<project>/.noodle/config.toml`

Later files override earlier files (`project` overrides `global`).

Legacy fallback is supported for both locations:
- `~/.noodle/config`
- `<project>/.noodle/config`

There is no `.noodlerc` file in Noodle.

## File Format

Noodle now parses real TOML.

- Strings must be quoted
- Durations are strings (for example, `"30s"`, `"72h"`)
- Numbers and booleans are unquoted
- `#` comments are supported

## Supported Structure

### Scalar Config (top-level or grouped)

These keys are supported either at top-level, or inside grouping tables like `[paths]`, `[cook]`, `[limits]`, `[budget]`/`[budgets]`:

- `project_dir = "/abs/path"`
- `brain_dir = "/abs/path/to/brain"`
- `noodle_dir = "~/.noodle/"`
- `autonomy = "manual" | "full" | "standard" | "safe"`
- `audit_interval = "72h"`
- `quality_interval = "24h"`
- `simmer = "30s"`
- `max_turns = 0`
- `history_window = 50`
- `review_threshold = 20.0`
- `cycle_budget = 0`
- `session_budget = 0`
- `cto_score_floor = 80`
- `retry_backoff = "2s"`
- `poll_interval = "500ms"`
- `prompts_dir = "/abs/path/to/prompts"` — override directory for prompt templates (optional; skips default project/global search when set)
- `verbose = true | false`

Notes:
- `cmdCook` overrides `project_dir` with the current working directory.
- `cmdCook` overrides `noodle_dir` with `~/.noodle`.
- `brain_dir` remains configurable and falls back to `<project>/brain` when unset.

### Default Model Policy

Preferred table form:

```toml
[model]
provider = "claude"
model = "claude-opus-4-6"
reasoning_level = "high"
```

Legacy flat keys still work:
- `model_provider`
- `model_name`
- `reasoning_level` / `reasoning`

### Worker Model Policy

Use this to control the default model policy for delegated worker sessions.

Preferred table form:

```toml
[worker]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "medium"
```

Legacy flat keys also work:
- `worker_model_provider`
- `worker_model_name` / `worker_model`
- `worker_model_reasoning_level` / `worker_model_reasoning`
- `worker_reasoning_level` / `worker_reasoning`

### Per-Entity Overrides

Preferred nested table form:

```toml
[entity.ceo]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "high"
```

Shorthand entity table is also supported:

```toml
[ceo]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "high"
```

Legacy flat pattern still works:
- `entity_model_<entity>_provider`
- `entity_model_<entity>_model`
- `entity_model_<entity>_reasoning_level`
- `entity_model_<entity>_reasoning`

### Per-Task Overrides

Preferred nested table form:

```toml
[task.execute]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "low"
```

Legacy flat pattern still works:
- `task_model_<task_class>_provider`
- `task_model_<task_class>_model`
- `task_model_<task_class>_reasoning_level`
- `task_model_<task_class>_reasoning`

### Model Capability Profiles

Define per-provider capability profiles to control concurrency limits and domain strengths:

```toml
[capabilities.claude]
strengths = ["frontend", "ui", "typescript", "react", "css"]
weaknesses = []
max_concurrent_tasks = 3
long_running = false

[capabilities.codex]
strengths = ["backend", "go", "rust", "systems", "planning"]
weaknesses = []
max_concurrent_tasks = 1
long_running = true
```

Default profiles are shipped for `claude` and `codex`. User config overrides these defaults.

Fields:
- `strengths` — array of domain/technology strings this provider excels at
- `weaknesses` — array of domain/technology strings this provider is weak at
- `max_concurrent_tasks` — maximum simultaneous sessions for this provider
- `long_running` — whether this provider handles long multi-step plans efficiently

### Per-Domain Model Overrides

Route tasks to specific models based on the inferred domain of their target files:

```toml
[domain.frontend]
provider = "claude"
model = "claude-opus-4-6"

[domain.backend]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "high"
```

Domain values: `frontend`, `backend`, `infra`, `mixed`. The assessor infers domain from target paths automatically.

Policy precedence (lowest to highest): Default → Entity → Domain → Task.

### Recommended Model Matrix Baseline

For Opus 4.6 + Codex 5.3 routing, use this baseline:

```toml
[model]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "high"

[domain.frontend]
provider = "claude"
model = "claude-opus-4-6"
reasoning_level = "high"

[domain.backend]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "high"

[task.verify]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "xhigh"

[task.quality]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "xhigh"

[task.systemfix]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "xhigh"
```

Spark guidance:
- Use `gpt-5.3-codex-spark` for speed-first interactive coding loops.
- Avoid global `[task.execute]` Spark override unless you want all execute tasks (including frontend) to use Spark.
- Prefer adaptive router logic for selective Spark assignment (for example backend execute with low risk/high latency sensitivity).

Full matrix spec:
- `noodle/cook/model-routing-spec.md`

## Example

```toml
# ~/.noodle/config.toml
[cook]
autonomy = "standard"
audit_interval = "72h"
quality_interval = "24h"
verbose = false

[model]
provider = "claude"
model = "claude-opus-4-6"
reasoning_level = "high"

[worker]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "medium"

[entity.ceo]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "high"

[task.execute]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "low"

[capabilities.claude]
strengths = ["frontend", "ui", "typescript", "react", "css"]
weaknesses = []
max_concurrent_tasks = 3
long_running = false

[capabilities.codex]
strengths = ["backend", "go", "rust", "systems", "planning"]
weaknesses = []
max_concurrent_tasks = 1
long_running = true

[domain.frontend]
provider = "claude"
model = "claude-opus-4-6"

[domain.backend]
provider = "codex"
model = "gpt-5.3-codex"
reasoning_level = "high"
```

## Safe Edit Procedure

1. `mkdir -p ~/.noodle`
2. Backup if the file exists.
3. Edit `~/.noodle/config.toml`.
4. For project-only overrides, edit `<project>/.noodle/config.toml`.
5. Validate with: `cd noodle && go run . help`
6. Parse-test with `cook.LoadConfig("$HOME/.noodle/config.toml")`.

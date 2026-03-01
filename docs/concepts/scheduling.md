# Scheduling

Noodle uses an LLM-powered scheduling loop to decide what work to do, when to do it, and which agent handles it. The scheduler is itself a skill — it reads project state, evaluates task types, and writes work orders.

## The Loop

The scheduling loop runs continuously. Each cycle follows this sequence:

1. **Brief** — Noodle gathers the current project state into a mise (French: everything in its place). This snapshot includes the backlog, active agents, recent history, available capacity, and registered task types.
2. **Schedule** — The scheduling agent reads the mise and decides what to work on next. It writes its decisions as orders.
3. **Orders** — Noodle reads the new orders and validates them against the task type registry.
4. **Dispatch** — For each pending stage in an order, Noodle spawns a cook (an agent session) to do the work.
5. **Cook** — The agent executes in an isolated worktree, reads the skill prompt, and does the work.
6. **Merge** — When a cook finishes, its changes merge back to the main branch. Merges are serialized to avoid conflicts.

The loop repeats on a timer and reacts to file changes. When `orders-next.json` or `control.ndjson` changes on disk, Noodle runs a cycle immediately.

## Mise en Place

Before each scheduling decision, Noodle builds a mise — a snapshot of everything the scheduler needs to know. The mise is written to `.noodle/mise.json` and contains:

- **Backlog** — items from your backlog adapter (GitHub issues, linear tickets, or a local file)
- **Active summary** — how many agents are running, grouped by task type, status, and runtime
- **Resources** — total cook capacity and how many slots are available
- **Recent history** — outcomes of recently completed sessions
- **Task types** — all registered task-type skills and their schedule triggers
- **Routing** — default and per-tag provider/model configuration
- **Warnings** — any issues the scheduler should know about (stale config, failed registries)

The scheduling agent reads this file and makes decisions based on all of it. Because the mise is a plain JSON file, you can inspect it directly to understand what the scheduler sees.

## Orders

An order is a unit of work with one or more stages. The scheduling agent writes orders to `.noodle/orders-next.json`. Noodle promotes this file to `.noodle/orders.json` after validation.

```json
{
  "id": "order-abc",
  "title": "Fix login timeout bug",
  "rationale": "High-priority backlog item, affects production",
  "stages": [
    {
      "task_key": "execute",
      "prompt": "Fix the login timeout bug described in issue #42",
      "provider": "claude",
      "model": "claude-opus-4-6",
      "status": "pending"
    }
  ],
  "status": "active"
}
```

Each order has:

| Field       | Description                                              |
| ----------- | -------------------------------------------------------- |
| `id`        | Unique identifier                                        |
| `title`     | Human-readable summary                                   |
| `rationale` | Why the scheduler chose this work                        |
| `plan`      | Optional list of high-level steps                        |
| `stages`    | The pipeline of work to execute                          |
| `status`    | `active`, `completed`, or `failed`                       |

## Stages

A stage is a single step within an order. Orders can have multiple stages that execute in sequence or in parallel groups.

| Field       | Description                                              |
| ----------- | -------------------------------------------------------- |
| `task_key`  | Which skill to run (maps to a task-type skill name)      |
| `prompt`    | Instructions for the agent                               |
| `skill`     | Skill to invoke (alternative to task_key)                |
| `provider`  | LLM provider for this stage                              |
| `model`     | Model to use                                             |
| `runtime`   | Where to run: `process` (local), `sprites` (cloud), `cursor` |
| `group`     | Parallel execution group number                          |
| `status`    | Current lifecycle status                                 |

### Stage Lifecycle

Each stage moves through a pipeline:

```
pending → active → merging → completed
                           → failed
                           → cancelled
```

- **pending** — waiting to be dispatched
- **active** — a cook is running this stage
- **merging** — the cook finished and its worktree is being merged
- **completed** — work merged successfully
- **failed** — the cook failed or the merge failed
- **cancelled** — manually cancelled or superseded

Stages within the same group number run in parallel. Groups execute in order — group 0 must complete before group 1 starts.

## Task Types

A skill becomes a task type when it has a `schedule` field in its frontmatter. The loop discovers task types at startup and hot-reloads them when skill files change on disk.

The mise includes all registered task types so the scheduling agent knows what kinds of work it can create orders for. The agent matches task type triggers against current conditions and creates orders accordingly.

See [Skills](/concepts/skills) for details on the `schedule` field.

## Routing

Each stage specifies a provider, model, and runtime. The scheduler can route different stages to different configurations:

- Run complex analysis on a powerful model, then run tests on a cheaper one
- Execute locally for fast iteration, or on cloud VMs for parallel throughput
- Match the model to the task — use a coding model for implementation, a reasoning model for architecture

Default routing comes from `.noodle.toml`:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.fast]
provider = "claude"
model = "claude-sonnet-4-6"
```

The scheduling agent can override defaults per stage when writing orders.

## Concurrency

Multiple cooks run in parallel, each in its own git worktree. The `max_cooks` setting in `.noodle.toml` controls how many run at once.

```toml
[concurrency]
max_cooks = 4
```

Each cook works on an isolated branch. When a cook finishes, its changes enter a merge queue. Merges are serialized — only one worktree merges at a time — to prevent conflicts. If a merge fails, the stage is marked failed and can be retried.

This means you can run many agents in parallel without them stepping on each other. The bottleneck is the merge queue, which processes one at a time to keep the main branch clean.

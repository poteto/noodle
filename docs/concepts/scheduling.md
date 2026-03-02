# Scheduling

Noodle uses an agent-powered scheduling loop to decide what work to do, when to do it, and which agent handles it. The scheduler is itself a [skill](/concepts/skills) — it reads project state, evaluates task types, and writes work orders.

## The Loop

Each cycle follows this sequence:

1. **Brief** — Noodle gathers current project state into a mise. This snapshot includes the backlog, active agents, recent history, available capacity, and registered task types.
2. **Schedule** — The scheduling agent reads the mise and decides what to work on next. It writes decisions as orders to `.noodle/orders-next.json`.
3. **Validate** — Noodle reads the new orders and validates them against the task type registry, then promotes the file to `.noodle/orders.json`.
4. **Dispatch** — For each pending stage, Noodle spawns a cook (an agent session) to do the work.
5. **Cook** — The agent executes in an isolated worktree, reads the skill prompt, and does the work.
6. **Merge** — When a cook finishes, its changes merge back to the main branch. Merges are serialized to avoid conflicts.

The loop runs continuously. It repeats on a timer and reacts to file changes — when `orders-next.json` or `control.ndjson` changes on disk, Noodle runs a cycle immediately.

## Mise en Place

Before each scheduling decision, Noodle builds a mise — a snapshot of everything the scheduler needs. Written to `.noodle/mise.json`, it contains:

| Section          | Contents                                                                    |
| ---------------- | --------------------------------------------------------------------------- |
| `backlog`        | Items from your backlog adapter (GitHub issues, Linear tickets, local file) |
| `active_summary` | Running agents grouped by task type, status, and runtime                    |
| `resources`      | Total cook capacity and available slots                                     |
| `recent_history` | Outcomes of recently completed sessions                                     |
| `task_types`     | All registered task-type skills and their schedule triggers                 |
| `routing`        | Default and per-tag provider/model configuration                            |
| `warnings`       | Issues the scheduler should know about (stale config, failed registries)    |

Because the mise is plain JSON, you can inspect it directly to see exactly what the scheduler sees.

## Orders

An order is a unit of work with one or more stages. The scheduling agent writes orders to `.noodle/orders-next.json`.

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

| Field       | Description                        |
| ----------- | ---------------------------------- |
| `id`        | Unique identifier                  |
| `title`     | Human-readable summary             |
| `rationale` | Why the scheduler chose this work  |
| `plan`      | Optional list of high-level steps  |
| `stages`    | Pipeline of stages to execute      |
| `status`    | `active`, `completed`, or `failed` |

## Stages

A stage is a single step within an order.

| Field      | Description                                     |
| ---------- | ----------------------------------------------- |
| `task_key` | Which task-type skill to run                    |
| `prompt`   | Instructions for the agent                      |
| `skill`    | Skill to invoke (alternative to `task_key`)     |
| `provider` | Agent provider for this stage                   |
| `model`    | Model to use                                    |
| `runtime`  | Where to run: `process`, `sprites`, or `cursor` |
| `group`    | Parallel execution group number                 |
| `status`   | Stage lifecycle status                          |

### Stage Lifecycle

```
pending → active → merging → completed
                           → failed
                           → cancelled
```

- **pending** — waiting to be dispatched
- **active** — a cook is running this stage
- **merging** — cook finished, worktree is being merged
- **completed** — work merged successfully
- **failed** — cook or merge failed
- **cancelled** — manually cancelled or superseded

Stages within the same `group` number run in parallel. Groups execute in order — group 0 must complete before group 1 starts.

## Task Types

A skill becomes a task type when it has a `schedule` field in its frontmatter. The loop discovers task types at startup and hot-reloads them when skill files change on disk.

The mise includes all registered task types so the scheduling agent knows what work it can create orders for. See [Skills](/concepts/skills) for details on the `schedule` field.

## Routing

Each stage specifies a provider, model, and runtime. The scheduler can route different stages to different configurations:

- Run analysis on a powerful model, then run tests on a cheaper one
- Execute locally for fast iteration, or on cloud VMs for parallel throughput
- Match the model to the task — a coding model for implementation, a reasoning model for architecture

Default routing comes from `.noodle.toml`:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.fast]
provider = "claude"
model = "claude-sonnet-4-6"
```

The scheduling agent can override defaults per stage when writing orders. See [Runtimes](/concepts/runtimes) for where stages execute.

## Concurrency

Multiple cooks run in parallel, each in its own git worktree. The `max_cooks` setting controls how many run at once:

```toml
[concurrency]
max_cooks = 4
```

Each cook works on an isolated branch. When it finishes, its changes enter a merge queue. Merges are serialized — one at a time — to keep the main branch clean. If a merge fails, the stage is marked failed and can be retried.

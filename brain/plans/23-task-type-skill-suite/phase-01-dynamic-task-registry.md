Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 1: Dynamic Task Registry — task.toml Discovery

## Goal

Replace the hardcoded Go task type registry with file-based discovery. Any skill with a `task.toml` becomes a task type. Built-in and user-defined task types follow the same convention.

## Current State

- `internal/taskreg/registry.go` — hardcoded constants (`TaskKeyPlan`, `TaskKeyExecute`, `TaskKeyVerify`, etc.) with a static map
- Adding a new task type requires Go code changes
- No user extensibility

## Changes

### Define task.toml convention

Create the `task.toml` file format. Noodle reads this; it is never loaded into the agent's context window.

```toml
blocking = false
review = true
schedule_hint = "After a cook session completes successfully"
```

- `blocking` (bool) — prevents other cooks from running in parallel
- `review` (bool) — output goes through quality gate after completion
- `schedule_hint` (string) — one-line natural language guidance for the prioritize skill

### Update skill resolver

Extend `skill/resolver.go` to parse `task.toml` during skill discovery. The resolver already scans skill directories — add `task.toml` parsing alongside SKILL.md detection. Return discovered task types as structured data.

### Refactor registry

Replace the hardcoded registry in `internal/taskreg/registry.go` with a discovery-based approach:
- On startup, the skill resolver scans all skill directories
- Skills with `task.toml` are registered as task types
- Skills without `task.toml` are utility skills (never scheduled by the loop)
- Remove hardcoded constants (`TaskKeyPlan`, `TaskKeyVerify`, etc.)
- Remove `TaskKeyPlan` and `TaskKeyVerify` entirely — plan is user-invoked, verify is handled by execute

### Create task.toml for built-in task types

Add `task.toml` to each built-in task type skill:

| Skill | blocking | review | schedule_hint |
|-------|----------|--------|---------------|
| prioritize | true | false | "When the queue is empty, after backlog changes, or when session history suggests re-evaluation" |
| quality | false | false | "After a cook session completes, before reflecting" |
| execute | false | true | "When a planned backlog item is ready for implementation" |
| reflect | false | false | "After a quality-approved cook session completes" |
| meditate | false | false | "Periodically after several reflect cycles have accumulated" |
| oops | false | false | "When a build failure, test failure, or infrastructure error is detected" |
| debate | false | false | "When a plan phase or design decision needs structured validation" |

### Include discovered task types in mise brief

The mise builder should include the list of discovered task types (name, blocking, review, schedule_hint) in the brief so the prioritize skill knows what task types are available and when to schedule them.

## Data Structures

- `task.toml` — per-skill metadata file (see convention above)
- Discovered task type: `{ name, blocking, review, schedule_hint, skill_path }`
- Mise brief gains a `TaskTypes []TaskType` field

## Verification

- `make ci` passes
- Skills with `task.toml` are discovered as task types: `go test ./skill/...`
- Skills without `task.toml` are not registered as task types
- No hardcoded task type constants remain in `internal/taskreg/`
- Mise brief includes discovered task types
- User-defined task type (create a test skill with `task.toml`) is discovered alongside built-ins

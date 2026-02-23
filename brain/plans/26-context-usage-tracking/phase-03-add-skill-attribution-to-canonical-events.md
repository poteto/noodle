Back to [[plans/26-context-usage-tracking/overview]]

# Phase 3: Add skill attribution to session metadata

## Goal

Attribute each session's context usage to the skill/task-type that spawned it. This enables per-skill aggregation downstream.

## Design note: reuse existing fields

`DispatchRequest` already has `Skill string` (line 16) and `TaskKey string` (line 23). `dispatchMetadata` already writes `Skill` to `spawn.json`. Reuse these names — do not introduce `SkillName`/`TaskType` duplicates.

## Changes

- **`dispatcher/dispatch_metadata.go`** — Add `TaskKey string` to `dispatchMetadata` so it's persisted in `spawn.json` alongside the existing `Skill` field.

- **`internal/sessionmeta/sessionmeta.go`** — Add fields to `Meta`:
  - `Skill string` — from `spawn.json` (already persisted)
  - `TaskKey string` — from `spawn.json` (added above)

- **`monitor/derive.go`** — Populate `Meta.Skill` and `Meta.TaskKey` from `spawn.json`. The monitor already reads session directories — add spawn metadata reading.

## Data structures

- `dispatchMetadata` gains `TaskKey string`
- `Meta` gains `Skill string` and `TaskKey string`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Field threading from spawn → meta |

## Verification

- `go test ./monitor/... ./dispatcher/...` — existing tests pass
- New test: spawn.json with skill + task key → meta.json has both populated
- New test: missing spawn.json → `Skill` and `TaskKey` are empty (graceful)

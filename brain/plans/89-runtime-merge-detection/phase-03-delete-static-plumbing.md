Back to [[plans/89-runtime-merge-detection/overview]]

# Phase 3 — Delete static `CanMerge` plumbing

## Goal

Remove the `Permissions` struct and `CanMerge` field from every layer: frontmatter parsing, task registry, mise types, schema docs, and task type summaries.

## Changes

**`skill/frontmatter.go`**:
- Delete `Permissions` struct (lines 29-31)
- Delete `CanMerge()` method (lines 33-38)
- Remove `Permissions` field from `NoodleMeta` struct

**`skill/frontmatter_test.go`**:
- Remove `CanMerge` assertions from `TestParseFrontmatterComplete`, `TestParseFrontmatterPartialNoodle`, `TestParseFrontmatterMergePermissionTrue`
- Remove `permissions:` blocks from test YAML fixtures
- Delete test cases that only existed to test merge permission parsing

**`skill/resolver_test.go`**:
- Remove `permissions: merge:` from test YAML fixtures (lines 222-223, 278-279)

**`internal/taskreg/registry.go`**:
- Remove `CanMerge` field from `TaskType` struct
- Remove `CanMerge` assignment in `NewFromSkills`

**`internal/taskreg/registry_test.go`**:
- Remove `Merge: boolPtr(false)` from test fixtures
- Remove `CanMerge` assertions

**`mise/types.go`**:
- Remove `CanMerge bool` from `TaskTypeSummary`

**`internal/schemadoc/specs.go`**:
- Remove `task_types[].can_merge` entry

**`loop/task_types.go`**:
- Remove `CanMerge: tt.CanMerge` from `registryToTaskTypeSummaries()`

**`cmd_mise.go`** (if it references `CanMerge`):
- Remove any references

## Data structures

- `Permissions` struct deleted
- `CanMerge` field removed from `TaskType` and `TaskTypeSummary`

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

```
go test ./skill/... ./internal/taskreg/... ./loop/... ./mise/...
go vet ./...
```

- Confirm no compile errors from removed fields
- Confirm all tests pass after removing `CanMerge` assertions
- Confirm `Permissions` struct is fully gone (grep for `Permissions` in skill package)

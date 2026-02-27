---
id: 74
created: 2026-02-27
status: done
---

# Vestigial Queue Cleanup

## Context

The order system replaced the flat queue months ago (plan 49). The event system replaced the audit log (plan 66). But naming artifacts, legacy compat shims, and one dead code path remain scattered across the codebase. Per [[principles/subtract-before-you-add]], removal creates a simpler substrate. Per [[principles/migrate-callers-then-delete-legacy-apis]], compat layers should not outlive their migration window.

## Scope

**In scope:**
- Delete `OrderID2` camelCase compat shim in `server/server.go` (update UI caller if needed)
- Delete legacy colon-separated format parser in `internal/snapshot/snapshot.go`
- Replace hand-rolled `slicesEqual` with `slices.Equal` from stdlib
- Rename 3 misnamed files (`loop/queue.go`, `loop/queue_audit_test.go`, `internal/orderx/queue.go`)
- Update 5 docs/skills that reference `.noodle/queue.json` (README, INSTALL, PHILOSOPHY, debugging skill, brain note)
- Fix vestigial "queue item IDs" description in status schema
- Rename `QueueDepth` field and CLI output in `cmd_status.go`
- Rename ~20 test function names/comments that still say "queue"

**Out of scope:**
- Refactoring single-implementation interfaces (test seams, not vestigial)
- Loop struct reorganization (organizational, not naming debt)
- Config complexity (separate concern)
- Archived plan files referencing "queue" (historical records)

## Constraints

- CLAUDE.md: "No backward compatibility by default" — delete compat, don't add new shims
- The UI may send `orderId` (camelCase) to the control endpoint — check and update the UI caller in the same phase as deleting `OrderID2`
- File renames require updating all import paths and test runner references
- `slices.Equal` requires Go 1.21+ (already satisfied)

## Applicable Skills

- `testing` — verify test renames don't break anything
- `go-best-practices` — standard patterns for file organization

## Phases

Ordered per subtract-before-you-add: delete first, then rename, then update docs.

- [x] [[archived_plans/74-vestigial-queue-cleanup/phase-01-delete-legacy-compat-code]] -- Delete compat shims and dead code
- [x] [[archived_plans/74-vestigial-queue-cleanup/phase-02-rename-vestigial-files]] -- Rename misnamed Go files
- [x] [[archived_plans/74-vestigial-queue-cleanup/phase-03-update-docs-skills-and-brain]] -- Update documentation references
- [x] [[archived_plans/74-vestigial-queue-cleanup/phase-04-rename-go-symbols-and-test-names]] -- Rename Go symbols and test functions

## Verification

```bash
go build ./... && go test ./... && go vet ./...
sh scripts/lint-arch.sh
grep -ri 'queue\.json\|QueueDepth\|queue_audit\|OrderID2' --include='*.go' --include='*.md'
```

The final grep should return zero matches (excluding archived plans and this plan file).

---
id: 97
created: 2026-03-02
status: active
---

# Adapter Schema Validator

Back to [[plans/index]]

## Context

Adapters are shell scripts that output NDJSON backlog items. Today, `ParseBacklogItems` in `adapter/sync.go` hard-fails on the first malformed line — one bad item kills the entire backlog sync. Users with broken adapters get no backlog and no feedback about what's wrong.

The fix: lenient parsing that skips bad items, collects per-item warnings, and surfaces those warnings through three channels — UI, backend logs, and the scheduler prompt. The scheduler can then create a fix task for the broken adapter.

## Scope

**In scope:**
- Lenient NDJSON parsing with per-item warnings (skip bad items, continue)
- Warning propagation: adapter → mise builder → loop state → UI snapshot
- Warning injection into the scheduler prompt
- Adapter docs page update with validation behavior

**Out of scope:**
- Validating Extra fields (pass-through by design)
- New CLI commands (`noodle adapter validate` would be a separate item)
- Changing the adapter output format or adding new required fields

## Constraints

- `ParseBacklogItems` signature changes from `([]BacklogItem, error)` to `([]BacklogItem, []string, error)` — all callers (runner, mise builder, fixture tests) must update **in the same phase** to keep the code compilable at every commit
- Warnings are ephemeral per cycle — regenerated each mise.Build(), not accumulated across cycles
- Scanner buffer increased to 1 MiB; `bufio.ErrTooLong` is a per-item warning, not fatal. Only true I/O errors return `(nil, nil, err)` — callers must not use partial results alongside errors
- Warning text is template-controlled (`fmt.Sprintf` with Go stdlib error descriptions), never raw adapter output — prevents prompt injection via crafted NDJSON
- LoopState is the existing pub/sub mechanism between loop and server — use it for dynamic warnings
- Warning changes must trigger the file-write → fsnotify → WS broadcast chain (add warnings to `stampStatus` equality check)
- Scheduler prompt follows the existing pattern: `buildSchedulePrompt()` takes typed parameters, not a grab-bag
- Warning dedup in server uses a single `mergeWarnings` helper (fresh slice allocation + `sort.Strings` + `slices.Compact`). Never sort shared slices in-place — concurrent HTTP/WS paths would race
- `lastMiseWarnings` cleared at cycle start so fatal cycles don't leave stale warnings

## Alternatives considered

**A. Lenient parsing with warnings (chosen)** — skip bad items, collect warnings, surface in UI/logs/scheduler. Non-blocking: adapter keeps working with valid items. Matches the todo description ("raise a warning").

**B. Strict validation with full rejection** — reject entire output on any violation. Simpler, but blocks scheduling entirely on adapter errors. Too aggressive.

**C. Separate validation CLI command** — `noodle adapter validate` for dev-time checking. Useful but doesn't catch runtime regressions or auto-create fix tasks. Could be a follow-up.

## Applicable skills

- `go-best-practices` — Go patterns, testing, error handling
- `testing` — TDD workflow, fixture tests

## Phases

1. [[plans/97-adapter-schema-validator/phase-01-lenient-parsing]] — Lenient parsing with warnings + all caller migration (adapter, runner, builder, tests)
2. [[plans/97-adapter-schema-validator/phase-02-ui-warnings]] — Surface warnings in UI via LoopState (stampStatus trigger, sorted dedup)
3. [[plans/97-adapter-schema-validator/phase-03-scheduler-injection]] — Inject warnings into scheduler prompt (capped, %q-escaped)
4. [[plans/97-adapter-schema-validator/phase-04-docs]] — Update adapter docs with validation behavior

## Verification

After all phases:
- `go test ./...` passes
- `go vet ./...` clean
- `pnpm build` succeeds
- `pnpm check` passes (if available)

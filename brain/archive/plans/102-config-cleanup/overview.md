---
id: 102-config-cleanup
created: 2026-03-01
status: completed
---

# Config Surface Cleanup (Todos 102-107)

Back to [[plans/index]]

## Context

Noodle's `.noodle.toml` exposes config knobs that are either unused, redundant with scheduler decisions, or internal implementation details. Before launch, shrink the config surface to only what users genuinely need to control.

## Scope

**In scope (6 todos):**

- **102** — Remove `routing.tags.*` from config. Scheduler decides routing per stage.
- **103** — Remove `[recovery]` section (`max_retries`), delete unused internal retry fallback (`defaultRetryMaxAttempts`), and remove all implicit auto-retry paths (including dispatch state-machine retry branches). Scheduler handles retry decisions via events and explicit control commands (`requeue` / `add-stage`).
- **104** — Remove `stuck_threshold` entirely; hardcode `poll_interval` to 1s.
- **105** — Remove ticket staleness tracking. Scheduler handles stuck stages via events.
- **106** — Rename `max_cooks` to `max_concurrency` in config and all Go code.
- **107** — Remove `max_completion_overflow`, `merge_backpressure_threshold`, `shutdown_timeout` from config. Hardcode sensible defaults. Shutdown: SIGTERM + 2s hard deadline.

**Out of scope:**
- `routing.defaults` stays — users set their default provider/model
- `[adapters]`, `[skills]`, `[agents]`, `[runtime]`, `[server]` sections untouched
- No new config fields added

## Constraints

- **Compatibility-first parse behavior.** Removed/unknown keys and invalid values must not hard-fail startup. Parse boundary logic must emit clear warnings naming the field and applied fallback, then continue with defaults.
- **Error messages describe failure state.** Per CLAUDE.md: "session not found", not "session must exist."
- **Boundary discipline.** Config warning/fallback normalization stays at the parse boundary in `config/parse.go`. Hardcoded defaults live where the value is consumed.
- **Scheduler-owned retries.** The loop should not auto-retry based on internal retryability classes; it should emit failure context and let the scheduler choose the next action.

### Alternatives considered

**Approach A (chosen): Sequential per-todo phases.** Each todo is one phase. Clean Git history, each phase independently shippable.

**Approach B: Batch config struct changes.** One big phase for all struct changes, another for all logic changes. Fewer files touched per phase but harder to review and revert.

**Approach C: Group by file.** One phase per affected file. Maximizes file locality but mixes unrelated semantic changes.

Approach A wins because each todo has different risk profiles (unused field deletion vs behavior change) and sequential phases give clear rollback points.

## Applicable Skills

- `go-best-practices` — Go conventions for the implementation phases
- `testing` — Test updates and new test cases
- `noodle` — Project conventions and config patterns

## Phases

1. [[plans/102-config-cleanup/phase-01-remove-recovery]]
2. [[plans/102-config-cleanup/phase-02-remove-ticket-staleness]]
3. [[plans/102-config-cleanup/phase-03-remove-stuck-detection]]
4. [[plans/102-config-cleanup/phase-04-remove-routing-tags]]
5. [[plans/102-config-cleanup/phase-05-remove-monitor-section]]
6. [[plans/102-config-cleanup/phase-06-rename-max-cooks]]
7. [[plans/102-config-cleanup/phase-07-hardcode-concurrency-internals]]
8. [[plans/102-config-cleanup/phase-08-shutdown-behavior]]

## Additional Cleanup Checklist

- This checklist is an index only; phase docs are authoritative for implementation details.
- **Retry behavior (Phase 1):** remove loop auto-retry branches on dispatch failures (`loop/cook_spawn.go`, `loop/schedule.go`) and any retry-only gate surface (`internal/mode/gate.go`, `docs/concepts/modes.md`) so retries are scheduler-directed.
- **Dispatch package cleanup (Phase 1, non-runtime-critical):** clean up `internal/dispatch` retry helpers/branches if they remain unused by runtime code paths.
- **Recovery config/docs (Phase 1):** remove all `[recovery]` references across config, docs, generator output, fixtures, and loop tests.
- **Routing tags (Phase 4):** remove all `routing.tags` plumbing in config/runtime/schema/docs/examples, including skill docs (`.agents/skills/noodle/SKILL.md`) and example docs (`examples/multi-skill/README.md`).
- **Monitor config/docs (Phases 2-5):** remove all `monitor.*` references from config/docs/generator output, including skill docs (`.agents/skills/noodle/SKILL.md`).
- **Concurrency rename (Phase 6):** complete `max_cooks` → `max_concurrency` across config/runtime/control/status/snapshot/UI/docs/examples/tests (including generated docs/tests).
- **Concurrency internals (Phase 7):** delete `max_completion_overflow` and `merge_backpressure_threshold` config surface, hardcode consumption defaults at runtime call sites, and update affected tests.
- **Shutdown behavior (Phase 8):** delete `shutdown_timeout` config surface and enforce fixed SIGTERM→2s→SIGKILL shutdown semantics in loop/dispatcher code paths.
- **Keep explicit retry controls:** retain scheduler/human-triggered controls such as `requeue` / `add-stage` as the supported retry mechanism.

## Verification

**Per-phase gate:** `pnpm check` (runs the full suite including build, vet, lint, and tests). This is the primary verification for every phase — `go build`/`go vet` alone are insufficient as they miss docs, schema, and UI integration.

After all phases:

```sh
pnpm check          # full suite — required gate
```

**E2e smoke test (after all phases):** `noodle start` → enqueue an order → cook runs to completion → `Ctrl-C` exits within ~2s. This proves the full config-parse → loop → cook → shutdown path works end-to-end with the reduced config surface.

**Config migration behavior:** old configs containing removed keys should still start, with explicit warnings and defaults applied.

Config surface should be:
- `[routing.defaults]` — provider, model
- `[concurrency]` — max_concurrency (renamed from max_cooks)
- `[adapters]`, `[skills]`, `[agents]`, `[runtime]`, `[server]` — unchanged

Removed sections: `[recovery]`, `[monitor]`, `routing.tags.*`
Removed fields from `[concurrency]`: max_completion_overflow, merge_backpressure_threshold, shutdown_timeout

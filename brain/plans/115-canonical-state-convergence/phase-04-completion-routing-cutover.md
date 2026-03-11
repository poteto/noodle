Back to [[plans/115-canonical-state-convergence/overview]]

# Phase 4 — Completion Routing Cutover

## Goal

Route live and adopted session completions, pending-review parking, and review control transitions through canonical records and reducer events instead of direct `orders.json` or `pending-review.json` mutation.

## Changes

- **`loop/cook_completion.go`** — convert `StageResult` handling into canonical completion/review routing; keep stage-message and scheduler shells, but stop editing orders directly
- **`internal/dispatch/dispatch.go`** — `RouteCompletion()` becomes the one completion-routing primitive
- **`internal/state/state.go` + `internal/reducer/reducer.go`** — add canonical pending-review state so supervised completion and review decisions have a first-class home in canonical state
- **`loop/orders.go` / `loop/state_orders.go` / `loop/pending_review.go`** — remove direct advancement/failure/review parking mutations once callers migrate
- **`loop/cook_spawn.go` + `loop/control_review.go`** — migrate dispatch-start failure, approval, request-changes, and reject paths off `failStage` and direct legacy file mutation

## Data structures

- `dispatch.CompletionRecord` — normalized completion input for live and adopted sessions
- canonical pending-review record embedded in `state.State`, sufficient for restart recovery, snapshot projection, approval, request-changes, and reject
- canonical event payloads for `stage_completed`, `stage_failed`, `stage_review_parked`, `stage_review_approved`, `stage_review_changes_requested`, and `stage_review_rejected`
- explicit failure event mapping for completion failure, dispatch-start failure, request-changes, and reject

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | High-risk lifecycle cutover with scheduler/review semantics |

## Verification

### Static
- `pnpm test:smoke`
- `go test ./internal/dispatch/... ./loop/... ./dispatcher/...`
- `go vet ./loop/... ./dispatcher/...`

### Runtime
- prove `stage_message` blocking/non-blocking behavior still matches current semantics
- prove `stage_yield` still overrides non-clean exits correctly
- prove live completions and adopted-session completions advance identically
- prove supervised completion can park for review without any direct legacy order or pending-review mutation
- prove schedule-stage special handling still removes/completes schedule orders without double-advancing
- prove approval, request-changes, and reject all operate through canonical review state and survive restart correctly
- prove dispatch-start failure and review-driven failure paths now reach canonical transitions instead of direct legacy mutation
- rerun `pnpm test:smoke` after completion/review cutover and treat unexpected failures as blockers

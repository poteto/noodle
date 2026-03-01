# Phase 1: Startup Boundary Classification

Back to [[plans/83-error-recoverability-taxonomy/overview]]

## Goal

Make `start` boundary outcomes explicit and typed: `abort`, `prompt-repair`,
or `warning-only`.

## Changes

- Classify startup/config/repair paths in command boundary code
- Encode intentional stop-for-repair as a typed recoverable boundary outcome
- Keep command behavior unchanged while making classification observable

## Data structures

- `StartBoundaryOutcome`
- `StartFailureEnvelope`
- `RepairPromptOutcome`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Mechanical boundary wiring in start/config paths |

## Verification

### Static

- Startup boundary code compiles with typed outcome wrappers
- Existing start command tests pass with updated assertions
- `go test ./... && go vet ./...`

### Runtime

- Invalid config yields classified hard startup failure
- Repairable diagnostics trigger classified prompt-repair outcome
- Web server/browser best-effort path remains warning-only

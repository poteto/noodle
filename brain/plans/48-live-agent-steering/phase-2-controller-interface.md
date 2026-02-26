Back to [[plans/48-live-agent-steering/overview]]

# Phase 2 — Controller Interface

## Goal

Define the `AgentController` abstraction that Claude and Codex transports implement. Extend `Session` to expose it. This phase is types and interfaces only — no protocol implementation yet.

## Changes

**New file: `dispatcher/controller.go`**

- `AgentController` interface:
  - `SendMessage(ctx context.Context, prompt string) error` — send a user message (blocks until delivered, not until turn completes)
  - `Interrupt(ctx context.Context) error` — request the agent to stop its current turn (blocks until acknowledged or timeout)
  - `Steerable() bool` — whether this controller supports live steering (false for noop)
- `noopController` — returns `ErrNotSteerable` from `SendMessage`/`Interrupt`, returns `false` from `Steerable()`

**Modify: `dispatcher/types.go`**

Add `Controller() AgentController` to the `Session` interface.

**Modify all Session implementations:**

- `processSession` (Phase 1) — returns `noopController` initially, replaced in Phases 3/4
- `spritesSession` — returns `noopController` (sprites out of scope)
- `adoptedSession` (in reconcile) — returns `noopController`
- Any test doubles — return `noopController`

This is an interface migration. Every implementation must be updated in this phase, even if just returning the noop. The steer rewrite (Phase 6) checks `Steerable()` to decide between live steering and kill+respawn fallback.

## Data Structures

- `AgentController` — interface
- `noopController` — fallback implementation
- `ErrNotSteerable` — sentinel error

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` — mechanical interface propagation.

## Verification

### Static
- `go build ./...` — all implementations satisfy the extended `Session` interface
- `go vet ./...`

### Runtime
- Unit test: `noopController.SendMessage` returns `ErrNotSteerable`
- Unit test: `noopController.Steerable()` returns `false`
- Existing test suite passes unchanged

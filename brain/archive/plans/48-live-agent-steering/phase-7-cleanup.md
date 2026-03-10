Back to [[plans/48-live-agent-steering/overview]]

# Phase 7 — Cleanup

## Goal

Remove tmux-specific code now that direct process management is the default. Subtract before you add — this phase is pure deletion.

## Changes

**Delete: `dispatcher/tmux_dispatcher.go`** (327 lines)
- Entire file — replaced by `process_dispatcher.go`

**Delete: `dispatcher/tmux_session.go`** (526 lines)
- Entire file — replaced by `process_session.go`

**Delete: `dispatcher/tmux_command.go`** (139 lines)
- Shell pipeline construction is no longer needed
- Provider command building that's still useful should already be extracted into `process_dispatcher.go` by Phase 1

**Modify: `monitor/observer.go`**
- Delete `TmuxObserver` type and its `Observe()` method
- Keep `PidObserver`, `HeartbeatObserver`, `CompositeObserver`

**Modify: `loop/reconcile.go`**
- Delete any remaining tmux references (should be clean after Phase 5, but verify)

**Modify: `dispatcher/factory.go`**
- Remove tmux dispatcher import/registration if still present
- Consider renaming runtime key from `"tmux"` to `"local"` or `"process"` (config migration)

**Modify: test fixtures**
- Update any fixtures that assert on tmux command format
- Update any fixtures that test the old steer kill+respawn flow
- Remove tmux-specific test helpers

**Review: `prompt.txt` / `input.txt` files**
- These are still written to session directories for debugging/auditability
- Remove from the provider pipeline (prompt goes over stdin now) but keep writing them as debug artifacts

## Data Structures

No new types. Deletion only.

## Routing

Provider: `codex`, Model: `gpt-5.4` — mechanical deletion and test updates.

## Verification

### Static
- `go build ./...`
- `go test ./...`
- `go vet ./...`
- `sh scripts/lint-arch.sh`
- `grep -r "tmux" --include="*.go" .` — should return zero hits outside of test comments/docs

### Runtime
- Full test suite passes
- Manual: end-to-end with both providers — spawn, steer, complete
- Manual: verify no tmux dependency at runtime (`which tmux` can be absent)

Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 11: Decompose Dispatcher `Dispatch` Methods

## Goal

Break the two long `Dispatch` methods in the dispatcher package into focused sub-functions.

## Changes

### 11a: `ProcessDispatcher.Dispatch` (117 lines, `process_dispatcher.go:56-173`)

Mixed concerns: validation, worktree resolution, directory creation, command building, process starting, metadata writing.

Extract:
- `resolveWorktreeForDispatch(req) (string, error)` — worktree setup
- `prepareSessionDir(runtimeDir, sessionID, req) error` — create session dir, write prompt/config
- `startSessionProcess(cmd, sessionDir) (*claudeController, error)` — start and configure

### 11b: `SpritesDispatcher.Dispatch` (129 lines, `sprites_dispatcher.go:72-201`)

Mixed concerns: validation, session setup, git operations, sprite upload, command construction.

Extract:
- `prepareRemoteRepo(ctx, sprite, req) (string, error)` — push/clone
- `uploadPromptFile(ctx, sprite, localPath) error` — upload prompt to sprite
- `buildSpriteCommand(req, systemPrompt) (*sprites.Cmd, io.ReadCloser, error)` — command assembly

### Also: Extract the triple-nested select

In `dispatcher/session_base.go:98-115`, extract the non-obvious drop-oldest-on-full backpressure logic into a named method `tryPublishEvent(ev SessionEvent)` with a comment explaining the policy.

## Data Structures

No new types. Sub-functions are private methods on their respective dispatcher structs.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Dispatcher code has process management and git operations — needs judgment for clean split points.

## Verification

### Static
- `go test ./dispatcher/...` — all dispatcher tests pass
- `go vet ./dispatcher/...` — clean
- No function in modified files exceeds 60 lines

### Runtime
- `go test ./...` — full suite passes

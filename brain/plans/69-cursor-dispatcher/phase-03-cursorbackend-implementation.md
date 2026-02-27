Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 3: CursorBackend Implementation

**Routing:** `codex` / `gpt-5.3-codex` — mapping between two typed APIs

## Goal

Replace the `CursorBackend` stub with a real `PollingBackend` implementation backed by `CursorClient` from phase 2. Maps between Noodle's backend-agnostic types (`PollLaunchConfig`, `PollResult`, `ConversationMessage`) and Cursor's API types.

## Data structures

- `CursorBackend` struct — holds `*CursorClient`, default repository

## Changes

**`dispatcher/cursor_backend.go`** (rewrite)
- `NewCursorBackend(apiKey, baseURL, repository string) *CursorBackend` — constructor creates internal `CursorClient`
- `Launch(ctx, PollLaunchConfig) (LaunchResult, error)` — builds `CreateAgentRequest` from config (`Prompt` → `prompt.text`, `Repository` → `source.repository`, `Model` → `model`, `Branch` → `source.ref`), calls `client.CreateAgent`, returns `LaunchResult{RemoteID: agentID, TargetBranch: response.Target.BranchName}`
- `PollStatus(ctx, remoteID) (PollResult, error)` — calls `client.GetAgent`, maps `CursorStatus` → `RemoteStatus` (CREATING/RUNNING → running, FINISHED → completed, ERROR → failed, EXPIRED → expired), populates `PollResult.Branch` from `target.branchName`, `PollResult.Summary` from `summary`
- `GetConversation(ctx, remoteID) ([]ConversationMessage, error)` — calls `client.GetConversation`, maps `AgentMessage` → `ConversationMessage`
- `Stop(ctx, remoteID) error` — calls `client.StopAgent`
- `Delete(ctx, remoteID) error` — calls `client.DeleteAgent`
- Keep `var _ PollingBackend = (*CursorBackend)(nil)` compile check

**`dispatcher/cursor_backend_test.go`** (rewrite)
- Test with `httptest.NewServer` backing the `CursorClient`
- Test: Launch builds correct request body, returns `LaunchResult` with agent ID and target branch
- Test: PollStatus maps each Cursor status correctly
- Test: PollStatus populates Branch and Summary from response
- Test: GetConversation maps messages
- Test: Stop and Delete call correct endpoints
- Test: API error propagation

## Verification

### Static
- `go vet ./dispatcher/...`
- `CursorBackend` satisfies `PollingBackend` at compile time

### Runtime
- `go test ./dispatcher/... -run TestCursorBackend -race`

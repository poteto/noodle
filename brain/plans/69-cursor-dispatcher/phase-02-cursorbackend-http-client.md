Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 2: CursorBackend HTTP client

**Routing:** `claude` / `claude-opus-4-6` — API client design, error handling judgment

## Goal

Replace the `CursorBackend` stub with a real HTTP client that talks to the Cursor Cloud Agents API. Implement all `PollingBackend` interface methods plus a `FollowUp` method for phase 6.

## Data structures

- `CursorBackend` struct — holds HTTP client, base URL, API key, optional webhook config
- `CursorAgentResponse` — maps the Cursor API agent object (id, name, status, source, target, summary, createdAt)
- `CursorConversationResponse` — maps the conversation endpoint response (id, messages)
- `CursorLaunchRequest` — maps the POST /v0/agents request body

## Changes

**`dispatcher/cursor_backend.go` (rewrite)**

Replace the stub entirely:

- Constructor: `NewCursorBackend(config CursorBackendConfig) *CursorBackend` — accepts API key, base URL (default `https://api.cursor.com`), HTTP client (injected for testing).
- `Launch(ctx, PollLaunchConfig) (string, error)` — POST /v0/agents. Maps PollLaunchConfig fields: Prompt → prompt.text, Repository → source.repository, Branch → source.ref, Model → model (omit for auto). Sets target.autoCreatePr=false, target.branchName from config's branch naming convention. Returns the Cursor agent ID (bc_* format).
- `PollStatus(ctx, remoteID) (RemoteStatus, error)` — GET /v0/agents/{id}. Maps Cursor statuses: CREATING/RUNNING → RemoteStatusRunning, FINISHED → RemoteStatusCompleted, ERROR → RemoteStatusFailed, STOPPED → RemoteStatusFailed.
- `GetConversation(ctx, remoteID) ([]ConversationMessage, error)` — GET /v0/agents/{id}/conversation. Maps each message to ConversationMessage with role and text.
- `Stop(ctx, remoteID) error` — POST /v0/agents/{id}/stop.
- `Delete(ctx, remoteID) error` — DELETE /v0/agents/{id}.
- `FollowUp(ctx, remoteID, text string) error` — POST /v0/agents/{id}/followup. Not part of PollingBackend interface; called directly by the session/dispatcher when needed.

Auth: Basic auth with API key as username, empty password. Set on every request.

Error handling: Check HTTP status codes. 4xx → descriptive error with status code and response body. 5xx → retriable error (but let the caller decide retry policy). Parse Cursor error responses when available.

**`dispatcher/backend_types.go`**

Add `TargetBranch string` to `PollLaunchConfig` — the branch the remote agent should push results to. Backend-agnostic: any polling backend that pushes to a branch needs this.

**`dispatcher/cursor_backend_test.go` (rewrite)**

Replace stub tests with real behavior tests using `httptest.NewServer`:
- Launch: verify request body, auth header, return mock agent response
- PollStatus: each Cursor status maps to correct RemoteStatus
- GetConversation: returns parsed messages
- Stop: sends POST, returns success
- Delete: sends DELETE, returns success
- FollowUp: verify request body
- Error cases: 401 unauthorized, 404 not found, 500 server error

## Verification

### Static
- Compiles, passes vet
- `CursorBackend` still satisfies `PollingBackend` interface
- All tests pass: `go test ./dispatcher/... -race`

### Runtime
- Unit tests with httptest mock server cover all 6 API methods + error paths
- Verify Basic auth header format: `Authorization: Basic base64(apikey:)`

Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 2: Cursor HTTP Client

**Routing:** `codex` / `gpt-5.3-codex` — HTTP client against a documented REST API

## Goal

Implement a typed HTTP client for the Cursor Cloud Agent API (v0). This is a pure HTTP layer — no dispatcher logic. The client handles auth, serialization, and error mapping. Used by `CursorBackend` in phase 3.

## Data structures

- `CursorClient` struct — holds `*http.Client`, base URL, API key
- Request/response types mirroring Cursor's API schema: `CreateAgentRequest`, `CreateAgentResponse`, `AgentStatus`, `AgentConversation`, etc.
- `CursorStatus` string enum — `CREATING`, `RUNNING`, `FINISHED`, `ERROR`, `EXPIRED`

## Changes

**`dispatcher/cursor_client.go` (new)**
- `NewCursorClient(apiKey, baseURL string) *CursorClient` — constructor. Default base URL `https://api.cursor.com` if empty.
- `CreateAgent(ctx, CreateAgentRequest) (CreateAgentResponse, error)` — POST `/v0/agents`
- `GetAgent(ctx, agentID string) (AgentResponse, error)` — GET `/v0/agents/{id}`
- `GetConversation(ctx, agentID string) ([]AgentMessage, error)` — GET `/v0/agents/{id}/conversation`
- `StopAgent(ctx, agentID string) error` — POST `/v0/agents/{id}/stop`
- `DeleteAgent(ctx, agentID string) error` — DELETE `/v0/agents/{id}`
- Internal: `do(ctx, method, path, body) (*http.Response, error)` — applies Basic auth header (`base64(apiKey + ":")`), content-type, error response parsing.
- HTTP errors → `APIError` (from phase 1) with `Retryable` classification: 429/5xx are retryable, 401/403/404/410 are terminal.
- 429 responses: parse `Retry-After` header when present.

**`dispatcher/cursor_client_test.go` (new)**
- Tests using `httptest.NewServer` for each endpoint
- Test: successful agent creation returns ID and target branch
- Test: get agent returns status and summary
- Test: get conversation returns messages
- Test: stop/delete succeed
- Test: API error (401, 429, 500) → `APIError` with correct `Retryable` classification
- Test: 429 with `Retry-After` header → parsed duration available
- Test: malformed JSON response → error

## Verification

### Static
- `go vet ./dispatcher/...`
- No external dependencies beyond stdlib

### Runtime
- `go test ./dispatcher/... -run TestCursorClient -race`
- All httptest-based tests pass

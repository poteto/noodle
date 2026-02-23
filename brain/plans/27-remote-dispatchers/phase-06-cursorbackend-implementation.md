Back to [[plans/27-remote-dispatchers/overview]]

# Phase 6: CursorBackend implementation

**Routing:** `codex` / `gpt-5.3-codex` — coding against a clear REST API spec

## References

- Cursor Cloud Agent API: https://cursor.com/docs/cloud-agent/api/endpoints

## Goal

Implement `CursorBackend` that satisfies the `PollingBackend` interface using the Cursor Cloud Agent REST API. No external SDK — plain `net/http` with JSON marshaling.

## Data structures

- `CursorBackend` struct — holds HTTP client, `CursorConfig` (API key resolved from env var at request time), base URL, default repo URL
- `CursorStatus` — maps Cursor's status strings (CREATING, RUNNING, FINISHED, ERROR, EXPIRED) to `RemoteStatus`

## Changes

**`dispatcher/cursor_backend.go` (new)**

`CursorBackend.Launch`:
1. `POST /v0/agents` with body: `{prompt: {text: prompt}, source: {repository: repoURL}, model: model}`
2. Auth: Bearer token header
3. Return agent ID from response

`CursorBackend.PollStatus`:
1. `GET /v0/agents/{id}`
2. Map status: CREATING/RUNNING → Running, FINISHED → Completed, ERROR → Failed, EXPIRED → Expired
3. Return `RemoteStatus` + summary text

`CursorBackend.GetConversation`:
1. `GET /v0/agents/{id}/conversation`
2. Return `[]ConversationMessage` from response messages array

`CursorBackend.Stop`:
1. `POST /v0/agents/{id}/stop`

`CursorBackend.Delete`:
1. `DELETE /v0/agents/{id}`

All HTTP calls include timeout via context, retry on 429 with backoff, structured error handling for non-2xx responses.

## Verification

### Static
- Compiles, passes vet
- `var _ PollingBackend = (*CursorBackend)(nil)` compile-time check

### Runtime
- Boundary test with `httptest.Server`: verify `Launch` sends correct POST body and parses agent ID from response
- Boundary test: verify `PollStatus` maps each Cursor status string (CREATING, RUNNING, FINISHED, ERROR, EXPIRED) to the correct `RemoteStatus`
- Boundary test: verify auth header format (Bearer token for Cursor)
- Boundary test: verify 429 response triggers retry with backoff

No live API tests — the `PollingDispatcher` tests (Phase 4) cover the full dispatch path with mock backends.

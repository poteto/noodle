---
id: 69
created: 2026-02-26
status: ready
---

# Cursor Dispatcher

## Context

Noodle currently supports two dispatch runtimes: tmux (local) and sprites (remote VM). Both use streaming backends — the agent's stdout is piped and parsed for events in real time.

Cursor Cloud Agents expose a REST API with no streaming. You launch an agent, poll for status, and optionally receive a webhook on terminal state changes (FINISHED, ERROR). The agent works on a GitHub branch and pushes changes there — no SSH, no stdout, no process handle.

This plan adds Cursor as a third dispatch runtime. The workflow mirrors sprites: push worktree branch → launch remote agent → agent pushes to target branch → pull branch back and merge. The key difference is monitoring: instead of streaming stdout, we poll the Cursor API for status and conversation messages, with webhooks as a fast-track for completion detection.

## Scope

**In scope:**
- `CursorBackend` — real HTTP client replacing the existing stub (plan 27 phase 6)
- `cursorSession` — Session implementation with polling loop and event synthesis
- `CursorDispatcher` — dispatch flow, prompt composition, branch workflow
- Webhook receiver endpoint on Noodle's HTTP server with HMAC-SHA256 verification
- Follow-up support (send additional instructions to a running Cursor agent)
- Config extensions and factory wiring
- Extract shared git/sync utilities from sprites_dispatcher.go for reuse

**Out of scope:**
- Generic PollingDispatcher abstraction (only one polling backend exists; subtract-before-you-add)
- Cursor agent model selection UI (user configures in .noodle.toml)
- Session adoption for cursor sessions on loop restart (future todo if needed)
- Image prompt support (Cursor API supports images, but not needed for code tasks)

## Constraints

- **Cursor API auth**: Basic auth with API key as username. Key read from env var (default `CURSOR_API_KEY`).
- **Repository access**: Cursor uses its own GitHub App integration. The repo must be accessible to the user's Cursor account. No git token injection needed (unlike sprites).
- **Webhook reachability**: The webhook URL must be accessible from Cursor's servers. If Noodle runs on localhost, the user needs a tunnel (ngrok, cloudflare tunnel) or a publicly routable address. Polling works without webhooks as fallback.
- **Rate limits**: List Repositories is severely rate-limited (1/min, 30/hr). We never call it. List Agents and Get Agent Status have standard limits.
- **No MCP support**: Cursor Cloud Agents don't support Model Context Protocol yet.
- **Prompt composition**: Cursor has no system prompt API — skill content and preamble must be inlined into `prompt.text`.

## Alternatives considered

### 1. Generic PollingDispatcher wrapping PollingBackend (rejected)
Plan 27 phase 4 designed a generic `PollingDispatcher` that accepts any `PollingBackend`. With webhook + follow-up being Cursor-specific, the generic abstraction adds complexity for exactly one consumer. If a second polling backend appears (Devin, Windsurf), we can extract then. Per subtract-before-you-add: don't build abstractions for one user.

### 2. Webhook only, no polling (rejected)
Simpler dispatcher but fragile if webhook delivery fails. Webhooks only fire on terminal states (FINISHED, ERROR) — no live conversation progress during RUNNING. Polling is needed regardless for live monitoring.

### 3. File-based webhook notification (rejected)
Webhook handler writes status to a file in the session directory; polling loop checks the file. Fits the file-based architecture but adds unnecessary indirection. A direct channel notification from server → session is simpler and instant.

## Applicable skills

- `go-best-practices` — all phases (Go implementation)
- `testing` — all phases (test-first verification)
- `review` — after each phase

## API Reference

Cursor Cloud Agents API (`https://api.cursor.com`):
- `POST /v0/agents` — launch agent (prompt, source repo/branch, target branch, webhook config)
- `GET /v0/agents/{id}` — get agent status (CREATING, RUNNING, FINISHED, STOPPED)
- `GET /v0/agents/{id}/conversation` — get conversation messages
- `POST /v0/agents/{id}/followup` — send follow-up instruction
- `POST /v0/agents/{id}/stop` — pause agent
- `DELETE /v0/agents/{id}` — delete agent permanently
- Webhook: POST to configured URL on statusChange (ERROR, FINISHED). HMAC-SHA256 signature in `X-Webhook-Signature` header.

## Phases

- [[plans/69-cursor-dispatcher/phase-01-extract-shared-remote-dispatcher-utilities]]
- [[plans/69-cursor-dispatcher/phase-02-cursorbackend-http-client]]
- [[plans/69-cursor-dispatcher/phase-03-cursorsession-poll-loop-and-event-synthesis]]
- [[plans/69-cursor-dispatcher/phase-04-cursordispatcher-and-factory-wiring]]
- [[plans/69-cursor-dispatcher/phase-05-webhook-receiver]]
- [[plans/69-cursor-dispatcher/phase-06-follow-up-support]]

## Verification

```bash
go test ./dispatcher/... && go test ./server/... && go test ./config/... && go test ./loop/... && go vet ./...
```

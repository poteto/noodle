Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 7: Webhook Receiver Endpoint

**Routing:** `claude` / `claude-opus-4-6` — security (HMAC verification), integration with polling lifecycle

## Goal

Add an HTTP endpoint to Noodle's server that receives webhook POST requests from Cursor when an agent's status changes. Verifies the HMAC-SHA256 signature, identifies the matching polling session, and short-circuits the poll interval so the session detects completion immediately.

**Prerequisite:** Noodle must be accessible from the internet for Cursor to deliver webhooks. This works out of the box if Noodle runs on a machine with a public IP or behind a reverse proxy. For localhost users, a tunnel (ngrok, cloudflare tunnel, etc.) is required. Polling (phases 4-5) remains the primary mechanism regardless.

## Data structures

- `WebhookPayload` struct — `Event string`, `ID string`, `Status string`, `Summary string`, `Target struct{ BranchName, PrURL string }`
- `WebhookNotifier` interface — `Notify(agentID string, status RemoteStatus, branch string)` — allows the server to signal a pollingSession to re-poll immediately

## Changes

**`server/server.go`**
- Add route: `mux.HandleFunc("POST /api/webhooks/cursor", s.handleCursorWebhook)`
- `handleCursorWebhook(w, r)`:
  1. Read raw body
  2. Verify HMAC-SHA256 signature from `X-Webhook-Signature` header using configured secret
  3. Parse `WebhookPayload`
  4. Call `WebhookNotifier.Notify(payload.ID, mappedStatus, payload.Target.BranchName)`
  5. Return 200 OK (fast — don't block Cursor's delivery)

**`dispatcher/polling_session.go`** (extend)
- Add `Nudge()` method to pollingSession — signals the poll goroutine to re-poll immediately (send on a nudge channel that `pollLoop` selects on alongside the ticker)
- `WebhookNotifier` implementation that looks up pollingSession by remote ID and calls `Nudge()`

**`config/config.go`**
- Add `WebhookSecret string` to `CursorConfig` — configured in `.noodle.toml` as `runtime.cursor.webhook_secret`
- Add `WebhookURL string` to `CursorConfig` — the public URL of the Noodle server, passed to Cursor at launch time

**`dispatcher/cursor_backend.go`** (extend)
- If webhook URL and secret are configured, include `webhook.url` and `webhook.secret` in the `CreateAgentRequest`

**`server/server_test.go`**
- Test: valid webhook with correct HMAC → 200, notifier called with correct args
- Test: invalid HMAC → 401, notifier not called
- Test: missing signature header → 401
- Test: malformed body → 400

## Verification

### Static
- `go vet ./server/... ./dispatcher/...`

### Runtime
- `go test ./server/... -run TestCursorWebhook -race`
- `go test ./dispatcher/... -run TestPollingSessionNudge -race`
- Manual: configure webhook secret, launch agent, verify webhook delivery triggers immediate completion detection

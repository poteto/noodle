Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 7: Webhook Receiver Endpoint

**Routing:** `claude` / `claude-opus-4-6` — security (HMAC verification), integration with polling lifecycle

## Goal

Add an HTTP endpoint to Noodle's server that receives webhook POST requests from Cursor when an agent's status changes. Verifies the HMAC-SHA256 signature, identifies the matching polling session via the `SessionNotifier` (wired in phase 6), and nudges it to re-poll immediately.

**Prerequisite:** Noodle must be accessible from the internet for Cursor to deliver webhooks. This works out of the box if Noodle runs on a machine with a public IP or behind a reverse proxy. For localhost users, a tunnel (ngrok, cloudflare tunnel, etc.) is required. Polling (phases 4-5) remains the primary mechanism regardless.

## Data structures

- `WebhookPayload` struct — `Event string`, `ID string`, `Status string`, `Summary string`, `Target struct{ BranchName, PrURL string }`

## Changes

**`server/server.go`**
- Add `SessionNotifier` field to server options (interface from phase 4: `Nudge(remoteID string)`)
- Add route: `mux.HandleFunc("POST /api/webhooks/cursor", s.handleCursorWebhook)`
- `handleCursorWebhook(w, r)`:
  1. Read raw body (limit to 1MB max to prevent abuse)
  2. Verify HMAC-SHA256 signature from `X-Webhook-Signature` header using configured secret — use `hmac.Equal` for constant-time comparison
  3. If no secret configured or no notifier wired, return 404 (endpoint not active)
  4. Parse `WebhookPayload`
  5. Call `notifier.Nudge(payload.ID)` — no-op if session unknown (idempotent)
  6. Return 200 OK (fast — don't block Cursor's delivery)

**`dispatcher/cursor_backend.go`** (extend)
- If webhook URL and secret are configured in `CursorBackend`, include `webhook.url` and `webhook.secret` in the `CreateAgentRequest` at launch time

**`config/config.go`** (already updated in phase 6)
- `WebhookSecretEnv` follows env-key pattern (reads from env var, defaults to `CURSOR_WEBHOOK_SECRET`)
- Add `WebhookURL string` to `CursorConfig` — the public URL of the Noodle server, passed to Cursor at launch time

**`server/server_test.go`**
- Test: valid webhook with correct HMAC → 200, notifier.Nudge called with correct agent ID
- Test: invalid HMAC → 401, notifier not called
- Test: missing signature header → 401
- Test: malformed body → 400
- Test: unknown agent ID → 200 (idempotent, no error)
- Test: duplicate delivery (same webhook ID) → 200 (idempotent)
- Test: no secret configured → 404
- Test: no notifier wired → 404
- Test: body exceeds 1MB → 413

## Verification

### Static
- `go vet ./server/... ./dispatcher/...`

### Runtime
- `go test ./server/... -run TestCursorWebhook -race`
- Manual: configure webhook secret and URL, launch agent, verify webhook delivery triggers immediate re-poll and faster completion detection

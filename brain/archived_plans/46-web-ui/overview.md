---
id: 46
created: 2026-02-24
status: done
---

# Web UI

Replace the Bubble Tea TUI with a React/TypeScript web UI served from the Go binary. `noodle start` launches a local HTTP server, opens the browser, and streams state updates via SSE. The design direction is established in the `ui_prototype/` — a kanban board with Slack-style agent chat, not a port of the TUI's tab layout.

## Prototype

Static HTML/CSS/JS prototype at `ui_prototype/` (`pnpm dev` to run). Directional on styling and layout, not high fidelity — not every planned interaction is mocked. Kanban board layout with Poster theme (bold yellow, hard drop shadows, Outfit + DM Sans). Includes:
- Board with Queued → Cooking → Review → Done columns
- Slack-style chat detail panel (click any agent card)
- @mention autocomplete for cross-agent steering
- Remote agent indicator (cloud icon with host tooltip)
- Mock data: 6 agents, event streams, queue items

Not yet prototyped (design during implementation):
- Request Changes prompt UI on review cards (phase 8)
- Pause/resume control in header (phase 10)
- Task editor modal (phase 10)
- Keyboard shortcuts (phase 10)

## Scope

**In scope:**
- Go HTTP server with SSE streaming (`server/` package)
- Extract `tui/model_snapshot.go` logic into reusable `internal/snapshot/` package
- TanStack Start SPA in `ui/` directory (SPA mode, no SSR)
- Kanban board: Queued → Cooking → Review → Done columns
- Agent detail: Slack-style chat panel with event stream, steer input, @mention
- Task editor, pause/resume, new task creation
- Remote agent indicator (cloud icon with host)
- Go embeds built SPA output via `embed.FS`
- `noodle start` opens browser by default, `--headless` skips both TUI and server

**Out of scope:**
- Deleting the Go TUI (deferred)
- Authentication/remote access

## Constraints

- Go server uses stdlib `net/http` only — no external framework
- SSE over WebSocket — simpler, sufficient for unidirectional state streaming. Control commands use POST.
- SPA mode (no SSR) — TanStack Start configured as pure client-side. The Go binary serves static files.
- Single binary distribution — `embed.FS` bundles the built UI. No separate `npm start` in production.
- Cross-platform — server must work on macOS/Linux/Windows

### Alternatives considered

1. **WebSocket instead of SSE** — Bidirectional, but adds complexity. The data flow is already "server pushes state, client sends commands." SSE + REST maps cleanly. Chose SSE.
2. **Separate dev server (Vite) in production** — Simpler dev story but breaks single-binary distribution. Chose `embed.FS` for production, Vite proxy for dev.
3. **Next.js/Remix instead of TanStack Start** — Heavier, SSR-focused. We want a pure SPA embedded in a Go binary. TanStack Start in SPA mode is the lightest option with good routing.

## Applicable skills

- `go-best-practices` — Go server phases (1-3)
- `frontend-design` — All React component phases — Poster theme, bold typography, distinctive aesthetic
- `interaction-design` — Microinteractions, transitions, scroll animations, loading states across UI phases
- `testing` — Verification across all phases

## Phases

1. [[archived_plans/46-web-ui/phase-01-extract-snapshot-package]] — Extract snapshot types + add RemoteHost, WorktreePath, document enums
2. [[archived_plans/46-web-ui/phase-02-go-http-server-with-sse]] — HTTP server, SSE, control POST with ack, config endpoint
3. [[archived_plans/46-web-ui/phase-03-server-integration-and-browser-launch]] — Wire into `noodle start`, browser launch, embed.FS
4. [[archived_plans/46-web-ui/phase-04-typescript-project-scaffold]] — TanStack Start SPA scaffold (can parallel with 1-3)
5. [[archived_plans/46-web-ui/phase-05-shared-client-package-types-and-sse-hook]] — TS types, SSE hook, React Query, kanban column derivation (blocks 6-10)
6. [[archived_plans/46-web-ui/phase-06-feed-view]] — Kanban board with four columns + agent/queue/review cards (parallel with 7)
7. [[archived_plans/46-web-ui/phase-07-queue-view]] — Board header: title, stats, loop state, new task button (parallel with 6)
8. [[archived_plans/46-web-ui/phase-08-reviews-view]] — Review actions: merge/reject/request-changes in Review column (parallel with 9, 10)
9. [[archived_plans/46-web-ui/phase-09-session-detail-view]] — Slack-style chat panel with event stream, steer input, @mention (parallel with 8, 10)
10. [[archived_plans/46-web-ui/phase-10-controls-steer-pause-task-editor]] — Pause/resume, task editor modal, keyboard shortcuts (parallel with 8, 9)

## Verification

- `go test ./...` and `go vet ./...` after Go phases
- `npm run build` and `npm run typecheck` in `ui/` after TypeScript phases
- Manual: `noodle start` opens browser, shows live session data, controls work
- Integration: Go server starts, SSE stream delivers snapshots, POST `/api/control` affects loop state

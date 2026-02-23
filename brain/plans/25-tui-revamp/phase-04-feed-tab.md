Back to [[plans/25-tui-revamp/overview]]

# Phase 3: Feed Tab

## Goal

Implement the Feed tab — a scrollable stream of agent activity rendered as bordered cards. Each event from the NDJSON log becomes a card. This is the default tab.

## Changes

### `tui/feed.go` — Feed tab implementation

Consumes `Snapshot.EventsBySession` and renders events as Card components in reverse-chronological order. Consecutive events from same session group into a single card.

Key type: `FeedTab` with `items []FeedItem`, selection index, scroll offset. Methods: `SetSnapshot(snap)`, `Render(width, height int) string`.

`FeedItem` wraps: agent name, task type, timestamp, event lines, optional verdict data (for Phase 8), optional steer data.

**Steer cards:** When the human steers an agent (via steer overlay), the steering instruction appears in the feed as a card attributed to "Chef". Rendered in brand yellow (#fde68a) border to visually distinguish from agent-generated cards. Shows: `★ Chef → {agent-name}`, the steering message in quotes, and timestamp. This gives visibility into human interventions — the feed tells the full story of what happened and why.

### `tui/feed_item.go` — Feed item rendering

Renders a single feed card. Agent name colored by task type, timestamp right-aligned in dim, body with event details. Supports simple events and rich events (test results, file changes).

### `tui/model.go` — Wire feed tab

Route snapshot updates to feed. Auto-scroll to newest unless user has scrolled up.

### `tui/model_snapshot.go` — Extend snapshot

Add `FeedEvents []FeedEvent` merging events across all sessions into chronological stream.

Key type: `FeedEvent` struct: `SessionID`, `AgentName`, `TaskType`, `At`, `Label`, `Body`, `Category`.

Category includes `"steer"` — sourced from control commands. When a steer command is written, it becomes a FeedEvent with Category `"steer"`, the human's message as Body, and the target agent name.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go test ./tui/...` passes
- Test: feed renders cards for 3 agents with interleaved events
- Test: steer events render as brand-yellow Chef cards
- Test: auto-scroll disengages on manual scroll up

### Runtime
- Launch with active cooks — feed shows real-time agent activity as cards
- Cards have visible borders, agent names color-coded by task type
- j/k scrolls, auto-scroll on new events
- Empty feed shows placeholder

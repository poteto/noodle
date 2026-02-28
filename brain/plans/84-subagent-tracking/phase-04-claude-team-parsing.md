Back to [[plans/84-subagent-tracking/overview]]

# Phase 4: Claude Team Parsing

## Goal

Parse Claude team lifecycle (TeamCreate, SendMessage, teammate inbox messages) into canonical agent events so Noodle can track team members and their communication.

## Changes

**`parse/claude.go`** -- Detect team-related tool_use blocks in `case "assistant"`:

1. **TeamCreate tool_use**: Emit `EventAgentSpawn` with:
   - `AgentType` = `"team_lead"` or from `input.agent_type`
   - `AgentName` from `input.team_name`
   - Message includes `input.description`

2. **SendMessage tool_use**: Emit `EventAgentProgress` with:
   - `AgentName` from `input.recipient`
   - Message from `input.content`
   - `AgentType` from `input.type` (message, broadcast, shutdown_request)

   Note: `EventAgentMessage` is deferred. Team communication is tracked via `EventAgentProgress` for now — it shows activity without requiring a separate event pipeline.

3. **Agent tool_use with `team_name`**: When `input.team_name` is set, the Agent tool is spawning a teammate. Emit `EventAgentSpawn` with:
   - `AgentName` from `input.name`
   - `AgentType` from `input.subagent_type`
   - `TeamName` from `input.team_name`
   - `Steerable` = `true`

4. **Teammate messages in NDJSON**: Team inbox traffic appears in the parent's NDJSON as `type: "user"` messages with `teamName` field and `<teammate-message teammate_id="..." color="..." summary="...">` XML wrappers in `message.content`. Parse the wrapper attributes to extract teammate identity and summary, then emit `EventAgentProgress`.

   Note: `input.agent_type` on TeamCreate is typically null in practice — fall back to `"team_lead"` when absent.

**Note on teams and steerability:** Claude team members spawned via `Agent` with `team_name` are steerable (`Steerable: true`). Regular Claude sub-agents (no team) are non-steerable.

**Out-of-band ingestion:** Team inbox files (`~/.claude/teams/{name}/inboxes/{agent}.json`) live outside the NDJSON stream. The session manager polls or watches these files and feeds new messages into the event pipeline. This is wired in Phase 5 alongside Codex sub-agent file discovery.

## Data Structures

- Team inbox message: `{From, Text, Summary string, Timestamp time.Time, Read bool, Color string}` — the `Text` field may contain nested JSON (e.g., `idle_notification`, `shutdown_request`, `task_assignment`) that should be parsed for display
- Reuse existing `claudeContent` for tool_use input parsing

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- requires judgment about what team events to surface vs skip (e.g., idle_notification JSON in inbox text should be parsed, not shown as raw text).

## Verification

### Static
- `go test ./parse/...`
- Test cases:
  - TeamCreate tool_use -> EventAgentSpawn with team_name
  - SendMessage tool_use (type: message) -> EventAgentProgress
  - SendMessage tool_use (type: shutdown_request) -> EventAgentProgress
  - Agent tool_use with team_name set -> EventAgentSpawn with Steerable=true and TeamName
  - `<teammate-message>` NDJSON entries -> EventAgentProgress with teammate identity

### Runtime
- Round-trip: construct fixture NDJSON, parse, verify events
- Inbox file parsing is out-of-band (Phase 5 wires the discovery/ingestion)

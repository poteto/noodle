Back to [[plans/84-subagent-tracking/overview]]

# Phase 4: Claude Team Parsing

## Goal

Parse Claude team lifecycle from the parent session's in-band NDJSON stream so Noodle can track team members and their communication without depending on inbox-file ingestion.

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

4. **Teammate messages in NDJSON**: Team activity appears in the parent's NDJSON as `type: "user"` messages with `teamName` field and `<teammate-message teammate_id="..." color="..." summary="...">` XML wrappers in `message.content`. Parse the wrapper attributes to extract teammate identity and summary, then emit `EventAgentProgress`.

   Note: `input.agent_type` on TeamCreate is typically null in practice — fall back to `"team_lead"` when absent.
   Fallback rule: if wrapper parsing fails or wrapper format drifts, emit a plain `EventAgentProgress` with raw summary text and a parse-warning action event (do not silently drop). That warning must remain visible in the session event stream so protocol drift is operator-visible, not parser-local.

**Note on teams and steerability:** Claude team members spawned via `Agent` with `team_name` are steerable (`Steerable: true`). Regular Claude sub-agents (no team) are non-steerable.

**Out-of-band ingestion (follow-up):** Team inbox files (`~/.claude/teams/{name}/inboxes/{agent}.json`) stay out of scope for plan 84. V1 only consumes in-band NDJSON team visibility; inbox-file polling/ingestion belongs to [[plans/88-subagent-tracking-v2/overview]].

## Data Structures

- Reuse existing `claudeContent` for tool_use input parsing
- Teammate NDJSON wrapper fields: `{teammate_id, color, summary}` extracted from the in-band XML wrapper before emitting canonical events

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
  - malformed/unknown wrapper format -> fallback progress + warning event

### Runtime
- Round-trip: construct fixture NDJSON, parse, verify events
- Verify v1 does not depend on inbox-file polling or reads

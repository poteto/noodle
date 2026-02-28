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

2. **SendMessage tool_use**: Emit `EventAgentMessage` with:
   - `AgentName` from `input.recipient`
   - Message from `input.content`
   - `AgentType` from `input.type` (message, broadcast, shutdown_request)

3. **Agent tool_use with `team_name`**: When `input.team_name` is set, the Agent tool is spawning a teammate. Emit `EventAgentSpawn` with:
   - `AgentName` from `input.name`
   - `AgentType` from `input.subagent_type` + marker that this is a team member
   - Extra metadata: `team_name`

**New: `parse/claude_team.go`** (or inline in claude.go) -- Team inbox file parser:

A separate function `ParseTeamInbox(path string) ([]CanonicalEvent, error)` that reads a Claude team inbox JSON file (array of `{from, text, summary, timestamp}`) and converts each message into `EventAgentMessage` events. This is called from outside the NDJSON pipeline -- by the session manager or a file watcher -- since inbox files are not part of the NDJSON stream.

## Data Structures

- Team inbox message: `{From, Text, Summary string, Timestamp time.Time, Read bool}`
- Reuse existing `claudeContent` for tool_use input parsing

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- requires judgment about what team events to surface vs skip (e.g., idle_notification JSON in inbox text should be parsed, not shown as raw text).

## Verification

### Static
- `go test ./parse/...`
- Test cases:
  - TeamCreate tool_use -> EventAgentSpawn
  - SendMessage tool_use (type: message) -> EventAgentMessage
  - SendMessage tool_use (type: shutdown_request) -> EventAgentMessage
  - Agent tool_use with team_name set -> EventAgentSpawn with team metadata
  - ParseTeamInbox with mixed messages and idle_notification JSON

### Runtime
- Round-trip: construct fixture NDJSON, parse, verify events
- ParseTeamInbox: read fixture inbox JSON file, verify message extraction

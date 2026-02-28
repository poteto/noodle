Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 4: Context Panel and Session Metrics

## Goal

The right-side context panel shows contextual information for the selected channel: system status for scheduler, session metrics + stage pipeline + files touched for agent channels.

## Skills

Invoke `react-best-practices`, `ts-best-practices`, `interaction-design` before starting.

## Routing

- **Provider:** `codex` | **Model:** `gpt-5.3-codex`
- Rendering metrics from snapshot data is mechanical

## Changes

### Modify
- `ui/src/components/ContextPanel.tsx` — full implementation:
  - **Scheduler context:** system status (loop state, active cooks, total cost, warnings)
  - **Agent context:**
    - Metrics grid: tokens, cost, elapsed time, context window usage %
    - Stage pipeline rail: ordered stages with status dots (done/active/pending)
    - Progress bar: stage completion percentage
    - Files touched list (from event stream — read/edit/write badges)

### Create
- `ui/src/components/MetricCard.tsx` — reusable metric display (label + value + optional unit)
- `ui/src/components/StageRail.tsx` — vertical stage pipeline with status indicators

## Data Structures

- Derive files touched by scanning `EventLine[]` for label "Read"/"Edit"/"Write" and extracting file paths from body
- `FileTouched` — `{ path: string, action: "read" | "edit" | "write" }`

## Tests

- `ContextPanel.test.tsx` — renders system metrics for scheduler channel, renders session metrics for agent channel
- `StageRail.test.tsx` — renders stages with correct status indicators, highlights active stage

## Verification

### Static
- `pnpm tsc --noEmit` and `pnpm test` pass

### Runtime
- Select scheduler → context panel shows system metrics
- Select agent → context panel shows session cost, tokens, stage pipeline
- Stage rail highlights current active stage
- Metrics update live as SSE events arrive

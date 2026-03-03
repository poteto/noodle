Back to [[plans/114-agent-native-onboarding/overview]]

# Phase 4: Improved Empty States

## Goal

Replace the generic "No sessions" / "No events yet" empty states across the UI with helpful, contextual messages that guide new users toward their next action.

## Changes

**`ui/src/components/Dashboard.tsx`:**
- Replace the "No sessions" table body with a proper empty state component.
- Content: "No sessions yet. Noodle runs sessions when the scheduler creates orders from your backlog." + link to /onboarding ("Learn how the loop works").

**`ui/src/components/SchedulerFeed.tsx`:**
- Improve the existing empty states:
  - "No scheduler session found" → add context: "The scheduler starts when you run `noodle start` and have skills installed." + link to /onboarding (this is the actual first-run landing view)
  - "Bootstrapping schedule skill" → add: "This usually takes a few seconds."

**`ui/src/components/Sidebar.tsx`:**
- "Waiting for the scheduler" → "No orders yet. The scheduler will create orders once it reads your backlog." Keep it brief — sidebar space is tight.

**`ui/src/components/AgentFeed.tsx`:**
- "Session not found" is an error state, not an empty state — leave it as-is.

**`ui/src/components/ReviewList.tsx`:**
- "No pending reviews" → add: "Completed sessions in supervised mode appear here for review."

## Data Structures

- No new types. May extract a shared `EmptyState` component if the pattern repeats, but only if 3+ places use identical structure.

## Design Notes

Empty states are the UI's onboarding. Every empty state should answer two questions: "why is this empty?" and "what do I do about it?" The current ones answer neither.

Keep messages short — 1-2 sentences max. Link to `/onboarding` from both the scheduler feed and dashboard empty states — the scheduler feed at `/` is the actual first-run landing view.

**Guard against error masking:** Empty states must only render when data loaded successfully with zero results. If the data fetch failed or is still loading, show the appropriate error/loading state instead. Don't let "No sessions yet" mask a backend connection failure.

**Note:** SchedulerFeed and Sidebar independently infer bootstrap state via similar logic. This duplication is a known issue but out of scope for this phase — a shared bootstrap-state hook should be extracted later.

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (UX writing requires judgment)

## Verification

### Static
- `pnpm --filter noodle-ui build`
- `pnpm --filter noodle-ui test`

### Runtime
- Start Noodle with no sessions, verify each empty state shows the new message:
  - Dashboard: "No sessions yet..." with link
  - Scheduler feed: contextual message with link
  - Sidebar: "No orders yet..."
  - Reviews: "No pending reviews. Completed sessions..."
- Start a session, verify empty states disappear and normal content renders

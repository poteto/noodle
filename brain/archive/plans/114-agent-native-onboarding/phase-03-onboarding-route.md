Back to [[plans/114-agent-native-onboarding/overview]]

# Phase 3: /onboarding Route

## Goal

Create a dedicated `/onboarding` route in the web UI that explains what Noodle is, how the loop works, and what to do next. Shown to brand-new installs.

## Changes

**New file `ui/src/routes/onboarding.tsx`:**
- TanStack Router route at `/onboarding`
- Standalone page (no sidebar, no feed layout — this is a first-contact experience)
- Content sections:
  1. **What is Noodle** — one paragraph: skill-based agent orchestration, kitchen brigade model
  2. **How the loop works** — visual diagram or step list: schedule → execute → quality → reflect → merge
  3. **What you need** — checklist: skills (schedule + execute minimum), a backlog (`todos.md`), at least one agent CLI (claude or codex)
  4. **Next steps** — actionable links: "Go to dashboard" (→ /dashboard), "View the live feed" (→ /)
- Design: clean, minimal, centered content. Use existing design tokens/theme. No illustrations needed — text + structure is enough.

**`ui/src/routes/__root.tsx`:**
- Add a conditional layout: if the current route is `/onboarding`, render only `<Outlet />` (no Sidebar, no ActiveChannelProvider wrapping). Otherwise render the normal app chrome. This lets the onboarding page be a true standalone first-contact experience without restructuring the entire route tree.

**`ui/src/routeTree.gen.ts`:**
- Auto-generated when the route file is created (TanStack Router codegen).

## Data Structures

- No new types — this is a static content page.

## Design Notes

This page is for the very first visit. Once the user has sessions running, they'll live in the dashboard and feed views. The onboarding page is always accessible at `/onboarding` but isn't prominently linked after the first visit.

The scheduler feed and dashboard empty states (phase 4) will link to `/onboarding` for users who haven't seen it yet. The onboarding page itself doesn't need to detect "first visit" — it's always the same content.

Keep the content concise. This isn't documentation — it's orientation. The docs site has the deep content. This page answers "what is this thing I just installed and what do I do now."

Use the `frontend-design` skill for implementation — this is a first-impression page and design quality matters.

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (UI design requires judgment)

## Verification

### Static
- `pnpm --filter noodle-ui build` — compiles without errors
- `pnpm --filter noodle-ui test` — existing tests pass
- Route exists: navigating to `/onboarding` renders the page
- Verify no Sidebar renders on `/onboarding`

### Runtime
- Visual: screenshot the page, verify it's clean and readable
- Navigation: click "Go to dashboard" and "View the live feed" links, verify they work
- Responsive: check the page at narrow widths (mobile-ish)

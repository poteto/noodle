Back to [[archive/plans/52-diff-viewer-for-reviews/overview]]

# Phase 3: Client diff API hook

## Goal

Add TypeScript types and a React Query hook for fetching the diff from the new API endpoint. This is the client-side data layer that the DiffViewer component will consume.

## Changes

**`ui/src/client/types.ts`**
- Add `DiffResponse` interface: `{ diff: string; stat: string }`

**`ui/src/client/hooks.ts`** (where existing hooks like `useSessionEvents` live; `index.ts` only re-exports)
- Add `useReviewDiff(itemId: string)` hook:
  - Fetches `GET /api/reviews/${itemId}/diff`
  - Uses React Query (`useQuery`) with a cache key like `["review-diff", itemId]`
  - `staleTime: Infinity` — the diff doesn't change while the item is in review (the agent is done, worktree is parked)
  - Returns `{ data, isLoading, error }` — standard React Query pattern
- Follow existing hook patterns in the client module (check how `useSessionEvents` is structured)

## Data structures

- `DiffResponse { diff: string; stat: string }`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical TypeScript, follows existing hook patterns |

## Verification

### Static
- `cd ui && npx tsc --noEmit`

### Runtime
- Import the hook in a test component, verify it fetches and returns data
- Verify cache behavior: second render with same ID doesn't refetch

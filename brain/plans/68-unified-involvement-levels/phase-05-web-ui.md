Back to [[plans/68-unified-involvement-levels/overview]]

# Phase 5: Web UI

## Goal

Update the web UI to display current mode and let users change it. Replace all `autonomy` references in TypeScript.

## Changes

- **ui/src/client/generated-types.ts**: `autonomy: string` → `mode: string` in Snapshot type
- **ui/src/client/types.ts**:
  - `ControlAction` union: replace `"autonomy"` with `"mode"`, add `"dispatch"`
  - `ConfigDefaults`: `autonomy: string` → `mode: string`
- **ui/src/client/api.ts**: Update config response type if needed
- **ui/src/components/Board.tsx**: Display current mode as a label or badge in the header. Add a dropdown to switch between auto/supervised/manual — sends `{ action: "mode", value: "..." }` via the control API.
- **ui/src/test-utils.ts**: `buildSnapshot()` — `autonomy: "supervised"` → `mode: "supervised"`
- **ui/src/client/api.test.ts**: Update `fetchConfig` test data
- **ui/src/client/types.test.ts**: `emptySnapshot()` — `autonomy` → `mode`
- **ui/src/components/Board.test.tsx**: Update any `applyOptimistic` tests if the autonomy action was tested

## Data Structures

- TypeScript: `mode: "auto" | "supervised" | "manual"` in snapshot type

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — UI design judgment

## Verification

### Static
- `cd ui && npm run build && npm run typecheck`
- No remaining `autonomy` references in `ui/src/`

### Runtime
- Visual: launch app, verify mode badge displays correctly
- Visual: change mode via dropdown, verify control command fires and UI updates via SSE
- `cd ui && npm run test`

Back to [[plans/68-unified-involvement-levels/overview]]

# Phase 4: Web UI

## Goal

Update the web UI to display current mode and let users change it. Add a dispatch button for manual mode. Replace all `autonomy` references in TypeScript.

## Changes

- **ui/src/client/generated-types.ts**: Regenerate from Go types (this file is auto-generated — don't hand-edit). `autonomy: string` → `mode: string` in Snapshot type.
- **ui/src/client/types.ts**:
  - `ControlAction` union: replace `"autonomy"` with `"mode"`, add `"dispatch"`
  - `ConfigDefaults`: `autonomy: string` → `mode: string`
- **ui/src/client/api.ts**: Update config response type if needed
- **ui/src/components/Dashboard.tsx**: Display current mode as a label or badge in the header. Add a dropdown to switch between auto/supervised/manual — sends `{ action: "mode", value: "..." }` via the control API. Add a "Dispatch" button visible in manual mode — sends `{ action: "dispatch" }` via the control API.
- **ui/src/test-utils.ts**: `buildSnapshot()` — `autonomy: "supervised"` → `mode: "supervised"`
- **ui/src/client/api.test.ts**: Update `fetchConfig` test data
- **ui/src/client/types.test.ts**: `emptySnapshot()` — `autonomy` → `mode`
- **ui/src/components/TaskEditor.test.tsx**: Update `autonomy` references

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
- Visual: in manual mode, dispatch button appears and triggers dispatch
- `cd ui && npm run test`

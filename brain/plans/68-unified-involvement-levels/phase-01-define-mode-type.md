Back to [[plans/68-unified-involvement-levels/overview]]

# Phase 1: Define Mode type

## Goal

Add the `Mode` type, constants, and query methods to config. Purely additive — old fields are untouched.

## Changes

- **config/config.go**: Add constants `ModeAuto = "auto"`, `ModeSupervised = "supervised"`, `ModeManual = "manual"`. Add `Mode string \`toml:"mode"\`` field to `Config`. Add query methods:
  - `AutoMerge() bool` — true when `Mode == ModeAuto`
  - `AutoDispatch() bool` — true when `Mode != ModeManual`
  - `AutoSchedule() bool` — true when `Mode != ModeManual`
- **config/config_test.go**: Test all three query methods for each mode value (9 cases)

## Data Structures

- `ModeAuto`, `ModeSupervised`, `ModeManual` string constants
- `Config.Mode string`

## Routing

Provider: `codex` | Model: `gpt-5.3-codex`

## Verification

### Static
- `go build ./...`
- `go vet ./...`

### Runtime
- `go test ./config/...` — new query method tests pass

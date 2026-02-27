Back to [[plans/68-unified-involvement-levels/overview]]

# Phase 3: Pass mode through mise, update skills

## Goal

Expose `mode` to the schedule skill via mise.json and update skill documentation.

## Changes

- **mise/types.go**: Add `Mode string \`json:"mode"\`` to `Brief`
- **mise/builder.go**: Read mode from the loop's live config at build time, not from a startup copy. Either accept mode as a `Build()` parameter or hold a config reference.
- **loop/loop.go** (or call site): Pass `l.config.Mode` when building the brief
- **.agents/skills/schedule/SKILL.md**: Document `mode` field in mise.json. Note that supervised mode means all merges require human approval — scheduler should consider this when sizing batches. Use `skill-creator` skill.
- Run `go generate ./generate/...` to confirm SKILL.md still up-to-date after any generator changes in phase 1.

## Data Structures

- `Brief.Mode string \`json:"mode"\``

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — skill content needs judgment

## Verification

### Static
- `go build ./...`
- `go vet ./...`

### Runtime
- Unit test: `mise.Build()` includes `Mode` in output
- `go test ./mise/... ./generate/...`
- Verify `go generate ./generate/...` produces up-to-date SKILL.md

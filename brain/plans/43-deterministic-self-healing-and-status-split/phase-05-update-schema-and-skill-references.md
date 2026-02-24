Back to [[plans/43-deterministic-self-healing-and-status-split/overview]]

# Phase 5: Update schema and skill references

## Goal

Update all schema documentation, skill prompts, and injected references to reflect the queue.json / status.json split.

## Changes

### Schema docs (`internal/schemadoc/specs.go`)

- Remove `active[]`, `autonomy`, `loop_state` field docs from the queue schema (lines 121-124)
- Add a new `status` schema target documenting status.json fields
- Update queue schema description if needed

### Prioritize skill (`/.agents/skills/prioritize/SKILL.md`)

- Add a note that status.json exists and contains loop runtime state
- No changes to the skill's core instructions — it already only writes queue.json

### Prioritize prompt injection (`loop/prioritize.go`)

- If the prioritize prompt currently injects queue schema that includes loop state fields, those references are now gone (handled by schema doc changes above)
- Add a reference to status.json in the injected prompt context so the agent is aware of it

### `noodle schema` CLI

- `noodle schema queue` should no longer show loop state fields
- `noodle schema status` should work (new target)

## Data structures

- New `statusfile.Status` type registered in schemadoc targets

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical doc/string updates across known files |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- Run `noodle schema queue` — verify no active/loop_state/autonomy fields
- Run `noodle schema status` — verify it shows the new fields
- Run a prioritize cycle — verify the agent receives status.json context in its prompt

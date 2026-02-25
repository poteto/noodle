Back to [[archived_plans/38-resilient-skill-resolution/overview]]

# Phase 1: Remove dead skill config fields

**Routing:** `codex` / `gpt-5.3-codex` — mechanical deletion, clear scope

## Goal

Remove config fields that map skill names but are either unused at runtime or superseded by automatic discovery. This is the subtract-before-you-add foundation.

## Changes

**Remove `phases` map:**
- `config/config.go` — delete `Phases map[string]string` field (line 28), default setup (lines 181-184), `applyDefaultsFromMetadata` logic (lines 272-280)
- `cmd_debug.go` — remove phases from debug dump (lines 114-121)
- `config/config_test.go` — remove phases assertions
- `generate/skill_noodle.go` — remove phases from generated docs
- `loop/fixture_test.go` — remove any phases references in fixture plumbing (line 267 area)

**Remove `prioritize.skill`:**
- `config/config.go` — delete `PrioritizeConfig.Skill` field (line 54), default (lines 187, 290), validation (lines 416-418)
- `config/config_test.go` — remove all `prioritize.skill` assertions
- `generate/skill_noodle.go` — remove from generated docs
- `loop/prioritize.go` — change `nonEmpty(item.Skill, "prioritize")` to just hardcode `"prioritize"` as the skill name (line 52). This was already the behavior since config was never read at runtime.

**Keep:**
- `skills.paths` — actively used for resolver search paths
- `adapters[*].skill` — actively used for domain skill injection in execute tasks

**Grep sweep:** Run `rg "Phases|prioritize\.skill|PrioritizeConfig" --type go` to catch any call sites not listed above. The exploration found fixture test references that are easy to miss.

## Data structures

- `Config.Phases` deleted
- `PrioritizeConfig.Skill` deleted

## Verification

```bash
go test ./config/... && go test ./loop/... && go vet ./...
```

Verify: config with `phases` or `prioritize.skill` keys still parses without error (TOML ignores unknown keys). Existing `.noodle.toml` files don't break.

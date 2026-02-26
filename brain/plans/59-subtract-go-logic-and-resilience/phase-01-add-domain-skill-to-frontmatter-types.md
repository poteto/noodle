Back to [[plans/59-subtract-go-logic-and-resilience/overview]]

# Phase 1: Add domain_skill to frontmatter types

Covers: #64 (foundation)

## Goal

Add `domain_skill` field to the `NoodleMeta` struct so task type skills can declare which domain skill they need injected at dispatch time.

## Changes

- `skill/frontmatter.go` — Add `DomainSkill string` to `NoodleMeta`
- `skill/frontmatter_test.go` — Test parsing `domain_skill` from frontmatter YAML

## Data structures

`NoodleMeta` gains `DomainSkill string \`yaml:"domain_skill,omitempty"\``

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical type addition with clear spec |

## Verification

### Static
- `go test ./skill/...` passes
- `go vet ./skill/...` clean
- New test: parse frontmatter with `domain_skill: backlog` under `noodle:`, assert field populated
- New test: parse frontmatter without `domain_skill`, assert field empty (backward compat)

### Runtime
- N/A — type-only change, no runtime behavior yet

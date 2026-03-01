Back to [[plans/89-runtime-merge-detection/overview]]

# Phase 4 — Update skill docs and frontmatter

## Goal

Remove `permissions: merge:` from actual skill files and update the skill documentation template.

## Changes

**`.agents/skills/quality/SKILL.md`**:
- Remove `permissions:` block from frontmatter (was `permissions: { merge: false }`)
- The `noodle:` block retains `schedule:` and any other fields

**`generate/skill_noodle.go`**:
- Remove `permissions: merge: true` from the example frontmatter
- Remove the `noodle.permissions.merge` row from the field table
- Remove the description of merge=false behavior
- Remove the mode override note (mode gating is separate and doesn't need to reference a deleted field)
- Use `skill-creator` skill to validate the template changes

**`.agents/skills/noodle/SKILL.md`** (generated):
- Regenerate from template after updating `generate/skill_noodle.go`

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Use `skill-creator` skill when updating skill files.

## Verification

```
go generate ./generate/...
diff .agents/skills/noodle/SKILL.md  # confirm generated output matches
```

- Confirm no skill frontmatter anywhere references `permissions: merge:`
- Confirm quality skill still has valid `noodle:` frontmatter with `schedule:` field

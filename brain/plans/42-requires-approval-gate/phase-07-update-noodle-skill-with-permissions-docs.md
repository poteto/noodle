Back to [[plans/42-requires-approval-gate/overview]]

# Phase 7: Update Noodle Skill with Permissions Docs

## Goal

Restructure the auto-generated noodle skill to follow skill-creator guidelines and document the new `permissions` object. The current generated skill has too much inline content — config schema, adapter examples, hook setup, troubleshooting — that should be split into reference files per progressive disclosure.

**Before writing anything:** Read `.agents/skills/skill-creator/SKILL.md` and its `references/` directory to internalize the guidelines. Apply them when restructuring the template. Key principles:
- Context window is a public good — only include what agents actually need
- Progressive disclosure: SKILL.md body for core workflow, references for details
- Keep SKILL.md under 500 lines
- Don't include information the model already knows

## Changes

### Restructure the generated output

The current SKILL.md inlines everything (~237 lines). Split into:

```
.agents/skills/noodle/
├── SKILL.md              (core: what noodle is, CLI commands, task-type frontmatter)
└── references/
    ├── config-schema.md  (full config table, minimal example, routing tags)
    ├── adapters.md       (adapter setup, script writing, examples)
    └── hooks.md          (brain injection hook, settings.json setup)
```

### SKILL.md body — what stays

- Brief description of what Noodle is (remove the kitchen brigade description — the brigade roles are not part of Noodle's public interface, they're internal implementation details that vary by user configuration)
- CLI commands table
- Task-type skill frontmatter spec including the new `permissions` object
- Pointers to reference files for config, adapters, and hooks

### SKILL.md body — what goes to references

- Config table -> `references/config-schema.md`
- Minimal config example -> `references/config-schema.md`
- Routing tags -> `references/config-schema.md`
- Adapter setup, script writing, GitHub/Linear examples -> `references/adapters.md`
- Hook installation -> `references/hooks.md`
- Troubleshooting -> `references/config-schema.md` (it's config-related)

### SKILL.md body — what gets removed

- Kitchen brigade description ("Cook does the work, Quality reviews it") — these are user-defined task types, not fixed Noodle architecture. An agent using Noodle doesn't need to know about a specific brigade metaphor.
- Any information an LLM would already know (e.g., explaining what TOML is, what adapters are conceptually)

### Task-type frontmatter spec (stays in SKILL.md)

Document the `noodle:` frontmatter spec that agents need to write correctly:

```yaml
noodle:
  schedule: "When to schedule this task type"
  permissions:
    merge: true   # default — auto-merge on success
    # merge: false — park worktree for human approval
```

- `schedule` is required, describes when the prioritize skill should schedule this
- `permissions.merge` defaults to `true`
- `permissions.merge: false` parks the worktree for human review

### `generate/skill_noodle.go` — Template changes

The Go template currently emits a single SKILL.md. Restructure it to emit multiple files, or change the generator to also write reference files. Review `generate/cmd/gen-skill/main.go` to understand how the output path works, then either:
- **Option A:** Expand the generator to write SKILL.md + reference files
- **Option B:** Make reference files static (not generated) and only generate SKILL.md

Option B is simpler if the reference content doesn't depend on Go struct reflection. The config table does (it reflects `config.Config`), so the config schema reference needs to stay generated. Adapters and hooks are static content.

### `generate/skill_noodle.go` — `fieldDescriptions`

Update the `"autonomy"` entry from `"full, review, or approve"` to `"auto or approve"`.

### Regenerate

```sh
go generate ./generate/...
git diff .agents/skills/noodle/  # review all generated output
```

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — skill design decisions, template restructuring.

## Verification

```sh
go test ./generate/...
go generate ./generate/...
# Review generated output follows skill-creator guidelines:
# - SKILL.md under 500 lines
# - Reference files exist and are linked from SKILL.md
# - No kitchen brigade description
# - permissions object documented
# - Config schema in references, not inlined
```

# Plan Templates

## Directory Structure

For plans with 3+ phases, create a directory:

```
brain/plans/42-mvp/
├── overview.md
├── phase-1-scaffold.md
├── phase-2-layout.md
├── phase-3-drawing.md
└── testing.md
```

Non-phase files (like `testing.md`) are fine alongside phases.

For small plans, a single file at `brain/plans/NN-plan-name.md` is fine.

## Overview File

Must include:
- **Context** — what problem this solves and why
- **Scope** — what's included, what's explicitly excluded
- **Constraints** — technical, platform, dependency, or pattern constraints
- **Applicable skills** — domain skills from Step 4 (list by name so implementers invoke them)
- **Phases** — ordered links to phase files: `[[plans/42-mvp/phase-1-scaffold]]`
- **Verification** — project-level verification commands (e.g. `go test ./...`)

## Phase Files

Each phase file must include:
- Back-link: `Back to [[plans/42-mvp/overview]]`
- **Goal** — what this phase accomplishes
- **Changes** — which files are affected and what changes, described at a high level
- **Data structures** — name the key types/schemas before logic (per foundational-thinking), but a one-line sketch is enough — don't write full definitions
- **Routing** — provider/model recommendation for this phase's execution:

| Phase type | Provider | Model | When |
|------------|----------|-------|------|
| Planning | `claude` or `codex` | `claude-opus-4-6` or `gpt-5.4-xhigh` | Writing plans, breaking down tasks |
| Review | `claude` or `codex` | `claude-opus-4-6` or `gpt-5.4-xhigh` | Code review, plan review, quality gates |
| Implementation | `codex` | `gpt-5.4` | Coding against a clear spec |
| Architecture / judgment | `claude` | `claude-opus-4-6` | Design decisions, complex refactoring |
| Skill creation | `claude` | `claude-opus-4-6` | Writing skills, brain notes |
| Scaffolding | `codex` | `gpt-5.4` | Boilerplate, mechanical transforms |

- **Verification** — static and runtime checks for this phase (see verification section below)

**Keep plans high-level.** Describe *what* and *why*, not *how* at the code level. A phase should read like a brief to a senior engineer: goals, boundaries, key types, and verification — not code snippets or pseudocode.

Order phases per the sequencing principle: infrastructure and shared types first, features after. Each phase should be independently shippable.

**Skill changes:** If a phase involves creating or updating a skill (any file in `.claude/skills/` or `.agents/skills/`), the phase must instruct the implementer to use the `skill-creator` skill during that phase.

## Verification Strategy

Every phase **must** have a verification section with both:

### Static
- Type checking passes
- Linting passes
- Go code follows project conventions (error handling, boundary discipline)
- Tests written and passing

### Runtime
- What to test manually (launch the app, exercise the feature path)
- What automated tests to write (unit, integration, e2e)
- Edge cases to cover
- For UI: visual verification via screenshot

Per prove-it-works principle: "it compiles" is not verification. Every phase must describe how to **prove** the change works.

### Common verification commands

- **Go**: `go test ./... && go vet ./...`
- **Lint**: `sh scripts/lint-arch.sh`

If verification requires launching the app or manual testing, the plan is not async-executable.

## CLI Scaffolding

Create the plan structure directly using file tools:

1. Create the plan directory: `brain/plans/NN-slug-name/`
2. Write `brain/plans/NN-slug-name/overview.md` with YAML frontmatter (`id`, `created`, `status: active`)
3. Write each phase file: `brain/plans/NN-slug-name/phase-01-name.md`, `phase-02-name.md`, etc.
4. Append `- [[plans/NN-slug-name/overview]]` to `brain/plans/index.md`
5. Add the plan wikilink `[[plans/NN-slug-name/overview]]` to the todo item in `brain/todos.md`

Fill in the overview and phase file content using Edit tools.

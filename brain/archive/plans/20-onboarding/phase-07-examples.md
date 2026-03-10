Back to [[plans/20-onboarding/overview]]

# Phase 7 — Example Projects

## Goal

Create an `examples/` directory with working example projects that demonstrate real Noodle workflows. Each example should be self-contained and runnable.

## Changes

- **`examples/minimal/`** — Minimal setup: `.noodle.toml` + one schedule skill + a simple backlog. Demonstrates the core loop: schedule → execute → commit.
- **`examples/multi-skill/`** — Multi-skill autonomous loop: custom task types (e.g., `deploy`, `test`), brain vault, quality review. Shows Noodle orchestrating multiple skill types.
- **`examples/README.md`** — Index of examples with descriptions and prerequisites

Each example includes:
- `.noodle.toml` with minimal config
- `.agents/skills/` with the skills needed
- `brain/todos.md` with sample backlog items
- `README.md` explaining what the example demonstrates and how to run it

## Data structures

None — config files and skill markdown only.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.4` | Mechanical file creation against a clear spec |

## Verification

### Static
- Each example's `.noodle.toml` parses without errors (validate with `noodle skills list` from the example directory)
- Skill files have valid frontmatter

### Runtime
- Run `noodle start` in each example directory — it bootstraps and starts the loop
- The minimal example produces at least one completed order
- Examples are referenced from the docs site getting-started guide

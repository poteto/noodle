Back to [[plans/97-adapter-schema-validator/overview]]

# Phase 4: Update Adapter Docs

## Goal

Document the validation behavior so adapter authors know what's checked, what produces warnings, and how warnings surface.

## Changes

### `docs/concepts/adapters.md`

Add a "Validation" section after the Schema section. Cover:

- What gets validated: each NDJSON line must be valid JSON, `id` and `title` are required non-empty strings
- What happens on failure: invalid items are skipped, valid items still process. Warnings are generated per bad item.
- Where warnings surface: web UI (warnings panel), backend logs, and the scheduler prompt (which may create a fix task)
- Example warning messages: `"line 3: invalid JSON: unexpected end of JSON input"`, `"line 5: missing required field title"`

Keep it short — the existing schema table already defines the contract. The validation section just explains what happens when the contract is broken.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Prose update, clear spec |

## Verification

### Static
- Docs render correctly (no broken links, markdown well-formed)

### Runtime
- N/A — documentation only

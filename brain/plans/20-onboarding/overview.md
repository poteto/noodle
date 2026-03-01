---
id: 20
created: 2026-03-01
status: active
---

# Plan 20 — Onboarding & First-Run Experience

## Context

Noodle is approaching launch readiness. The codebase has a comprehensive README, a PHILOSOPHY.md, and deep architecture docs in `brain/`, but no structured docs site, no example projects, and no step-by-step getting-started guide. A developer who finds Noodle on GitHub needs to go from zero to a running autonomous loop in minutes — not by reading architecture docs, but by following a clear path.

## Scope

**In scope:**
- Docs site (VitePress on GitHub Pages)
- Lean README rewrite (quick-start focus, links to docs site for depth)
- Core concept documentation (skills, scheduling, brain, kitchen model)
- In-repo `examples/` directory with working example projects
- Getting-started tutorial (zero to running Noodle)
- Skill path documentation (`.agents/skills`, `.claude/skills`, custom paths)

**Out of scope:**
- Changing skill path defaults (current behavior is correct)
- Code changes to the bootstrap/first-run flow (works well as-is)
- Video tutorials, screencasts
- External examples repo
- API reference docs (auto-generated from code)

## Target audience

Developers already using Claude Code or Codex who want to automate with Noodle. Assumes familiarity with AI agents and skills. Focus on what Noodle adds: scheduling, orders, brain, kitchen model.

## Constraints

- Docs must be version-controlled in-repo (not a separate repo)
- GitHub Pages deployment via GitHub Actions
- Prose must sound human — use the `unslop` skill on all written content
- Docs site styling should match the Noodle web UI (shared design language — colors, typography, spacing). Reference `ui/` for design tokens. Use `frontend-design` skill for custom theme/CSS.
- All examples must actually work against the current codebase (verified by running them)

## Alternatives considered

**Docs site framework:**
1. **VitePress** — Vue-based, lightweight, excellent docs defaults, fast. De facto standard for OSS docs.
2. **Docusaurus** — React-based (matches app UI), heavier, more features than needed.
3. **Plain Jekyll via GitHub Pages** — Zero setup, limited flexibility, looks generic.

**Chosen: VitePress.** Lightest option with the best docs UX out of the box. Vue vs React doesn't matter — the docs site is a separate concern from the app UI.

## Reference

- [[plans/20-onboarding/reference-docs-site-mockup.html]] — HTML mockup of target docs site design (design tokens, layout, visual style). **Styling and layout reference only — page content is illustrative, not actual docs content.**

## Applicable skills

- `unslop` — Apply to all prose to remove AI writing patterns
- `frontend-design` — Custom docs site theme/styling
- `interaction-design` — Microinteractions, transitions, and polish (hover states, scroll behavior, code block interactions, nav transitions)
- `skill-creator` — If any skill files need updating

## Phases

1. [[plans/20-onboarding/phase-01-docs-scaffold]] — VitePress project setup + theme + nav structure
2. [[plans/20-onboarding/phase-02-readme]] — Existing docs audit + lean README rewrite
3. [[plans/20-onboarding/phase-03-config-reference]] — Configuration reference (ships early — concept docs link to it)
4. [[plans/20-onboarding/phase-04-concept-skills]] — Skills concept doc
5. [[plans/20-onboarding/phase-05-concept-scheduling]] — Scheduling concept doc
6. [[plans/20-onboarding/phase-06-concept-brain-modes]] — Brain & modes concept doc
7. [[plans/20-onboarding/phase-07-examples]] — Example projects
8. [[plans/20-onboarding/phase-08-getting-started]] — Getting-started tutorial
9. [[plans/20-onboarding/phase-09-deploy]] — GitHub Pages deployment

## Verification

- `pnpm --filter docs build` succeeds (docs site builds clean)
- All internal links resolve (VitePress dead-link detection)
- Examples run successfully against current Noodle binary
- `pnpm check` still passes (no regressions)
- Docs site deploys to GitHub Pages and renders correctly

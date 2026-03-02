Back to [[plans/20-onboarding/overview]]

# Phase 1 — Docs Site Scaffold

## Goal

Set up a VitePress project in `docs/` that builds and serves a documentation site with Noodle's visual identity. Includes the complete navigation structure, custom theme, and landing page. Interaction polish (`interaction-design` skill) should be applied incrementally here and in later phases — not a blocker for shipping.

## Changes

- **`docs/`** — New VitePress project (package.json, config, theme)
- **`docs/.vitepress/config.ts`** — Site config: title, description, nav, sidebar, `base` path for GitHub Pages (decide the hosting URL now — all dev-time links must match production)
- **`docs/.vitepress/theme/`** — Custom theme extending default VitePress theme. Style should match the Noodle web UI (colors, typography, spacing). Reference `ui/` source for design tokens. Use `frontend-design` skill for visual design. Apply `interaction-design` incrementally (hover states, nav transitions, code block interactions).
- **`docs/index.md`** — Landing page (hero section, feature highlights, quick links)
- **`package.json`** — Add `docs` workspace, add `docs:dev` and `docs:build` scripts
- **`docs/failure-message-policy.md`** — Migrate the existing file in `docs/` into the VitePress structure (or move to a `contributing/` section)

### Nav structure

Define the complete sidebar upfront — subsequent phases fill in content:

```
Getting Started          → /getting-started.md
Concepts
  Skills                 → /concepts/skills.md
  Scheduling             → /concepts/scheduling.md
  Brain                  → /concepts/brain.md
  Modes                  → /concepts/modes.md
Reference
  Configuration          → /reference/configuration.md
Examples                 → /examples.md
```

## Reference

- [[plans/20-onboarding/reference-docs-site-mockup.html]] — HTML mockup using Noodle UI tokens from `ui/src/app.css`. **Styling and layout reference only — page content is illustrative, not actual docs content.** Key design elements: near-black backgrounds (`#050505`/`#0a0a0a`/`#111111`), `#f5c518` yellow accent (active nav, keywords, TOC indicator, callout borders, blinking cursor), JetBrains Mono throughout, pixel borders (`#222222`/`#333333`), dot-grid content background, inverted yellow active nav links, left-rail TOC with yellow active dot.

## Data structures

- VitePress `UserConfig` — nav items, sidebar groups, theme config

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Design decisions for site structure and styling |

## Verification

### Static
- `pnpm --filter docs build` succeeds
- No TypeScript errors in config

### Runtime
- `pnpm --filter docs dev` serves the site locally
- Landing page renders with custom styling
- Navigation structure shows placeholders for all planned sections

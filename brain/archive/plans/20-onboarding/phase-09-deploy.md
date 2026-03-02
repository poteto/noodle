Back to [[plans/20-onboarding/overview]]

# Phase 9 — GitHub Pages Deployment

## Goal

Set up automated deployment of the docs site to GitHub Pages via GitHub Actions. The `base` path was already configured in Phase 1.

## Changes

- **`.github/workflows/docs.yml`** — New workflow:
  - Trigger: push to `main` (paths: `docs/**`)
  - Steps: install pnpm, build VitePress, deploy to GitHub Pages
  - Uses `actions/deploy-pages` or equivalent

- **Repository settings** — Document: enable GitHub Pages, set source to GitHub Actions

## Data structures

None — CI/CD configuration only.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.3-codex` | Mechanical CI config against a clear spec |

## Verification

### Static
- Workflow YAML is valid
- `pnpm --filter docs build` succeeds in CI

### Runtime
- Push to main triggers the workflow
- Docs site is accessible at the GitHub Pages URL
- All pages render correctly
- Links between pages work (relative paths correct for base path)

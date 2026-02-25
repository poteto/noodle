Back to [[archived_plans/46-web-ui/overview]]

# Phase 4: TypeScript Project Scaffold

## Goal

Set up the `ui/` directory with TanStack Start in SPA mode, Vite, TypeScript, and the dev toolchain.

## Changes

- **`ui/`** — Initialize TanStack Start project in SPA mode. Key config:
  - `app.config.ts` — TanStack Start config with SPA mode enabled
  - `vite.config.ts` — Vite config with proxy to Go server for `/api/*` in dev mode
  - `tsconfig.json` — Strict TypeScript
  - `package.json` — Dependencies: `@tanstack/react-start`, `@tanstack/react-router`, `@tanstack/react-query`, `react`, `react-dom`
- **Dev workflow** — In dev: run `npm run dev` (Vite on port 5173) + `noodle start` (Go server on port 3000). Vite proxies `/api/*` to Go. In production: `npm run build` outputs to `ui/dist/`, Go embeds it.
- **Root `.gitignore`** — Add `ui/node_modules/`, `ui/dist/`
- **Scripts** — `npm run dev`, `npm run build`, `npm run typecheck`

## Data structures

None — this is project scaffolding.

## Routing

Provider: `codex` | Model: `gpt-5.3-codex` — mechanical scaffolding.

## Verification

### Static
- `npm run typecheck` passes
- `npm run build` produces output in `ui/dist/`

### Runtime
- `npm run dev` starts Vite, serves a blank page at localhost:5173
- API proxy works: `curl localhost:5173/api/snapshot` proxies to Go server

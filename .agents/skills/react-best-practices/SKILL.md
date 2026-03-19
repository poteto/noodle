---
name: react-best-practices
description: >-
  React performance optimization guidelines. Use when writing, reviewing, or refactoring React
  components, hooks, effects, data fetching, or bundle optimization. Includes useEffect best
  practices — when NOT to use effects and better alternatives. Triggers: "when writing React code",
  "reviewing a React PR", "why is this component re-rendering", "reviewing code for performance
  issues", "optimizing bundle size".
---

# React Best Practices

Performance optimization guide for client-side React applications. Contains 47 rules across 8 categories, prioritized by impact. Adapted from Vercel Engineering's guidelines, filtered for client-only React + Vite (no Next.js / SSR).

## When to Apply

Reference these guidelines when:
- Writing new React components or hooks
- Reviewing code for performance issues
- Refactoring existing React code
- Optimizing bundle size or render performance
- Writing or reviewing `useEffect` usage

## Rule Categories by Priority

| Priority | Category | Impact | Prefix |
|----------|----------|--------|--------|
| 1 | Eliminating Waterfalls | CRITICAL | `async-` |
| 2 | Bundle Size Optimization | CRITICAL | `bundle-` |
| 3 | Client-Side Data Fetching | MEDIUM-HIGH | `client-` |
| 4 | Re-render Optimization | MEDIUM | `rerender-` |
| 5 | useEffect Patterns | MEDIUM | `useeffect-` |
| 6 | Rendering Performance | MEDIUM | `rendering-` |
| 7 | JavaScript Performance | LOW-MEDIUM | `js-` |
| 8 | Advanced Patterns | LOW | `advanced-` |

## Quick Reference

### 1. Eliminating Waterfalls (CRITICAL)

- `async-defer-await` — Move await into branches where actually used
- `async-parallel` — Use Promise.all() for independent operations
- `async-dependencies` — Use promise chaining for partial dependencies
- `async-api-routes` — Start promises early, await late

### 2. Bundle Size Optimization (CRITICAL)

- `bundle-barrel-imports` — Import directly, avoid barrel files
- `bundle-conditional` — Load modules only when feature is activated
- `bundle-defer-third-party` — Load analytics/logging after mount
- `bundle-preload` — Preload on hover/focus for perceived speed

### 3. Client-Side Data Fetching (MEDIUM-HIGH)

- `client-event-listeners` — Deduplicate global event listeners
- `client-passive-event-listeners` — Use passive listeners for scroll/touch
- `client-swr-dedup` — Use SWR for automatic request deduplication
- `client-localstorage-schema` — Version and minimize localStorage data

### 4. Re-render Optimization (MEDIUM)

- `rerender-derived-state-no-effect` — Derive state during render, not effects
- `rerender-defer-reads` — Don't subscribe to state only used in callbacks
- `rerender-simple-expression-in-memo` — Avoid memo for simple primitives
- `rerender-memo-with-default-value` — Hoist default non-primitive props
- `rerender-memo` — Extract expensive work into memoized components
- `rerender-dependencies` — Use primitive dependencies in effects
- `rerender-move-effect-to-event` — Put interaction logic in event handlers
- `rerender-derived-state` — Subscribe to derived booleans, not raw values
- `rerender-functional-setstate` — Use functional setState for stable callbacks
- `rerender-lazy-state-init` — Pass function to useState for expensive values
- `rerender-transitions` — Use startTransition for non-urgent updates
- `rerender-use-ref-transient-values` — Use refs for transient frequent values

### 5. useEffect Patterns (MEDIUM)

- `useeffect-anti-patterns` — Common mistakes: derived state in effects, effect chains, notifying parents
- `useeffect-alternatives` — Decision tree and alternatives: derived state, key prop, event handlers, useSyncExternalStore

### 6. Rendering Performance (MEDIUM)

- `rendering-animate-svg-wrapper` — Animate div wrapper, not SVG element
- `rendering-content-visibility` — Use content-visibility for long lists
- `rendering-hoist-jsx` — Extract static JSX outside components
- `rendering-svg-precision` — Reduce SVG coordinate precision
- `rendering-conditional-render` — Use ternary, not && for conditionals
- `rendering-usetransition-loading` — Prefer useTransition for loading state

### 7. JavaScript Performance (LOW-MEDIUM)

- `js-batch-dom-css` — Batch DOM reads/writes to avoid layout thrashing
- `js-index-maps` — Build Map for repeated lookups
- `js-cache-property-access` — Cache object properties in loops
- `js-cache-function-results` — Cache function results in module-level Map
- `js-cache-storage` — Cache localStorage/sessionStorage reads
- `js-combine-iterations` — Combine multiple filter/map into one loop
- `js-length-check-first` — Check array length before expensive comparison
- `js-early-exit` — Return early from functions
- `js-hoist-regexp` — Hoist RegExp creation outside loops
- `js-min-max-loop` — Use loop for min/max instead of sort
- `js-set-map-lookups` — Use Set/Map for O(1) lookups
- `js-tosorted-immutable` — Use toSorted() for immutability

### 8. Advanced Patterns (LOW)

- `advanced-init-once` — Initialize app once per app load
- `advanced-event-handler-refs` — Store event handlers in refs
- `advanced-use-latest` — useEffectEvent for stable callback refs

## How to Use

Read individual reference files for detailed explanations and code examples:

```
references/async-parallel.md
references/rerender-derived-state-no-effect.md
references/useeffect-anti-patterns.md
```

Each reference file contains:
- Brief explanation of why it matters
- Incorrect code example with explanation
- Correct code example with explanation
- Additional context and references

# Sections

This file defines all sections, their ordering, impact levels, and descriptions.
The section ID (in parentheses) is the filename prefix used to group references.

Adapted from Vercel Engineering's React Best Practices for client-only React + Vite + Tauri.
Server-side / Next.js-specific rules have been removed.

---

## 1. Eliminating Waterfalls (async)

**Impact:** CRITICAL
**Description:** Waterfalls are the #1 performance killer. Each sequential await adds full network latency. Eliminating them yields the largest gains.

## 2. Bundle Size Optimization (bundle)

**Impact:** CRITICAL
**Description:** Reducing initial bundle size improves Time to Interactive and Largest Contentful Paint.

## 3. Client-Side Data Fetching (client)

**Impact:** MEDIUM-HIGH
**Description:** Automatic deduplication and efficient data fetching patterns reduce redundant network requests.

## 4. Re-render Optimization (rerender)

**Impact:** MEDIUM
**Description:** Reducing unnecessary re-renders minimizes wasted computation and improves UI responsiveness.

## 5. useEffect Patterns (useeffect)

**Impact:** MEDIUM
**Description:** Effects are an escape hatch. Most state updates, derived values, and user interactions should be handled without effects.

## 6. Rendering Performance (rendering)

**Impact:** MEDIUM
**Description:** Optimizing the rendering process reduces the work the browser needs to do.

## 7. JavaScript Performance (js)

**Impact:** LOW-MEDIUM
**Description:** Micro-optimizations for hot paths can add up to meaningful improvements.

## 8. Advanced Patterns (advanced)

**Impact:** LOW
**Description:** Advanced patterns for specific cases that require careful implementation.

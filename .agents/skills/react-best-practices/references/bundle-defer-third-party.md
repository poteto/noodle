---
title: Defer Non-Critical Third-Party Libraries
impact: MEDIUM
impactDescription: loads after initial render
tags: bundle, performance
---

## Defer Non-Critical Third-Party Libraries

**Impact: MEDIUM (loads after initial render)**

Analytics, logging, and error tracking don't block user interaction. Load them after the initial render.

**Incorrect: blocks initial bundle**

```tsx
import { Analytics } from '@vercel/analytics/react'

export default function App({ children }) {
  return (
    <>
      {children}
      <Analytics />
    </>
  )
}
```

**Correct: lazy-loaded after initial render**

```tsx
import { lazy, Suspense } from 'react'

const Analytics = lazy(() =>
  import('@vercel/analytics/react').then(m => ({ default: m.Analytics }))
)

export default function App({ children }) {
  return (
    <>
      {children}
      <Suspense fallback={null}>
        <Analytics />
      </Suspense>
    </>
  )
}
```

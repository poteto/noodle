---
title: Do not wrap a simple expression with a primitive result type in useMemo
impact: LOW-MEDIUM
impactDescription: wasted computation on every render
tags: rerender, performance
---

## Do not wrap a simple expression with a primitive result type in useMemo

**Impact: LOW-MEDIUM (wasted computation on every render)**

When an expression is simple (few logical or arithmetical operators) and has a primitive result type (boolean, number, string), do not wrap it in `useMemo`.

Calling `useMemo` and comparing hook dependencies may consume more resources than the expression itself.

**Incorrect:**

```tsx
function Header({ user, notifications }: Props) {
  const isLoading = useMemo(() => {
    return user.isLoading || notifications.isLoading
  }, [user.isLoading, notifications.isLoading])

  if (isLoading) return <Skeleton />
  // return some markup
}
```

**Correct:**

```tsx
function Header({ user, notifications }: Props) {
  const isLoading = user.isLoading || notifications.isLoading

  if (isLoading) return <Skeleton />
  // return some markup
}
```

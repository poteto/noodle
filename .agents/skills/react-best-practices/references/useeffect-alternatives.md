---
title: Better Alternatives to useEffect
impact: MEDIUM
impactDescription: eliminates unnecessary effects and render passes
tags: useeffect, hooks, derived-state, alternatives
---

## Better Alternatives to useEffect

### Table of Contents

- [Quick Reference](#quick-reference)
- [When You DO Need Effects](#when-you-do-need-effects)
- [When You DON'T Need Effects](#when-you-dont-need-effects)
- [Decision Tree](#decision-tree)
- [1. Calculate During Render (Derived State)](#1-calculate-during-render-derived-state)
- [2. useMemo for Expensive Calculations](#2-usememo-for-expensive-calculations)
- [3. Key Prop to Reset State](#3-key-prop-to-reset-state)
- [4. Store ID Instead of Object](#4-store-id-instead-of-object)
- [5. Event Handlers for User Actions](#5-event-handlers-for-user-actions)
- [6. useSyncExternalStore for External Stores](#6-usesyncexternalstore-for-external-stores)
- [7. Lifting State Up](#7-lifting-state-up)
- [8. Custom Hooks for Data Fetching](#8-custom-hooks-for-data-fetching)
- [Summary: When to Use What](#summary-when-to-use-what)

### Quick Reference

| Situation | DON'T | DO |
|-----------|-------|-----|
| Derived state from props/state | `useState` + `useEffect` | Calculate during render |
| Expensive calculations | `useEffect` to cache | `useMemo` |
| Reset state on prop change | `useEffect` with `setState` | `key` prop |
| User event responses | `useEffect` watching state | Event handler directly |
| Notify parent of changes | `useEffect` calling `onChange` | Call in event handler |
| Fetch data | `useEffect` without cleanup | `useEffect` with cleanup OR framework |

### When You DO Need Effects

- Synchronizing with **external systems** (non-React widgets, browser APIs)
- **Subscriptions** to external stores (use `useSyncExternalStore` when possible)
- **Analytics/logging** that runs because component displayed
- **Data fetching** with proper cleanup (or use framework's built-in mechanism)

### When You DON'T Need Effects

1. **Transforming data for rendering** - Calculate at top level, re-runs automatically
2. **Handling user events** - Use event handlers, you know exactly what happened
3. **Deriving state** - Just compute it: `const fullName = firstName + ' ' + lastName`
4. **Chaining state updates** - Calculate all next state in the event handler

### Decision Tree

```
Need to respond to something?
├── User interaction (click, submit, drag)?
│   └── Use EVENT HANDLER
├── Component appeared on screen?
│   └── Use EFFECT (external sync, analytics)
├── Props/state changed and need derived value?
│   └── CALCULATE DURING RENDER
│       └── Expensive? Use useMemo
└── Need to reset state when prop changes?
    └── Use KEY PROP on component
```

### 1. Calculate During Render (Derived State)

For values derived from props or state, just compute them:

```tsx
function Form() {
  const [firstName, setFirstName] = useState('Taylor')
  const [lastName, setLastName] = useState('Swift')

  // Runs every render - that's fine and intentional
  const fullName = firstName + ' ' + lastName
  const isValid = firstName.length > 0 && lastName.length > 0
}
```

**When to use**: The value can be computed from existing props/state.

### 2. useMemo for Expensive Calculations

When computation is expensive, memoize it:

```tsx
import { useMemo } from 'react'

function TodoList({ todos, filter }) {
  const visibleTodos = useMemo(
    () => getFilteredTodos(todos, filter),
    [todos, filter]
  )
}
```

**How to know if it's expensive**:
```tsx
console.time('filter')
const visibleTodos = getFilteredTodos(todos, filter)
console.timeEnd('filter')
// If > 1ms, consider memoizing
```

### 3. Key Prop to Reset State

To reset ALL state when a prop changes, use key:

```tsx
// Parent passes userId as key
function ProfilePage({ userId }) {
  return (
    <Profile
      userId={userId}
      key={userId}  // Different userId = different component instance
    />
  )
}

function Profile({ userId }) {
  // All state here resets when userId changes
  const [comment, setComment] = useState('')
  const [likes, setLikes] = useState([])
}
```

**When to use**: You want a "fresh start" when an identity prop changes.

### 4. Store ID Instead of Object

To preserve selection when list changes:

```tsx
// BAD: Storing object that needs Effect to "adjust"
function List({ items }) {
  const [selection, setSelection] = useState(null)

  useEffect(() => {
    setSelection(null) // Reset when items change
  }, [items])
}

// GOOD: Store ID, derive object
function List({ items }) {
  const [selectedId, setSelectedId] = useState(null)

  // Derived - no Effect needed
  const selection = items.find(item => item.id === selectedId) ?? null
}
```

**Benefit**: If item with selectedId exists in new list, selection preserved.

### 5. Event Handlers for User Actions

User clicks/submits/drags should be handled in event handlers, not Effects:

```tsx
// Event handler knows exactly what happened
function ProductPage({ product, addToCart }) {
  function handleBuyClick() {
    addToCart(product)
    showNotification(`Added ${product.name}!`)
    analytics.track('product_added', { id: product.id })
  }

  function handleCheckoutClick() {
    addToCart(product)
    showNotification(`Added ${product.name}!`)
    navigateTo('/checkout')
  }
}
```

**Shared logic**: Extract a function, call from both handlers:

```tsx
function buyProduct() {
  addToCart(product)
  showNotification(`Added ${product.name}!`)
}

function handleBuyClick() { buyProduct() }
function handleCheckoutClick() { buyProduct(); navigateTo('/checkout') }
```

### 6. useSyncExternalStore for External Stores

For subscribing to external data (browser APIs, third-party stores):

```tsx
import { useSyncExternalStore } from 'react'

function subscribe(callback) {
  window.addEventListener('online', callback)
  window.addEventListener('offline', callback)
  return () => {
    window.removeEventListener('online', callback)
    window.removeEventListener('offline', callback)
  }
}

function useOnlineStatus() {
  return useSyncExternalStore(
    subscribe,
    () => navigator.onLine
  )
}
```

### 7. Lifting State Up

When two components need synchronized state, lift it to common ancestor:

```tsx
// Instead of syncing via Effects between siblings
function Parent() {
  const [value, setValue] = useState('')

  return (
    <>
      <Input value={value} onChange={setValue} />
      <Preview value={value} />
    </>
  )
}
```

### 8. Custom Hooks for Data Fetching

Extract fetch logic with proper cleanup:

```tsx
function useData(url) {
  const [data, setData] = useState(null)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let ignore = false
    setLoading(true)

    fetch(url)
      .then(res => res.json())
      .then(json => {
        if (!ignore) {
          setData(json)
          setError(null)
        }
      })
      .catch(err => {
        if (!ignore) setError(err)
      })
      .finally(() => {
        if (!ignore) setLoading(false)
      })

    return () => { ignore = true }
  }, [url])

  return { data, error, loading }
}
```

### Summary: When to Use What

| Need | Solution |
|------|----------|
| Value from props/state | Calculate during render |
| Expensive calculation | `useMemo` |
| Reset all state on prop change | `key` prop |
| Respond to user action | Event handler |
| Sync with external system | `useEffect` with cleanup |
| Subscribe to external store | `useSyncExternalStore` |
| Share state between components | Lift state up |
| Fetch data | Custom hook with cleanup / framework |

Reference: [https://react.dev/learn/you-might-not-need-an-effect](https://react.dev/learn/you-might-not-need-an-effect)

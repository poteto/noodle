---
name: interaction-design
description: >
  Design and implement microinteractions, motion design, transitions, and user feedback
  patterns. Use when adding polish to UI interactions, implementing loading states, creating
  gesture-based interactions, designing hover/focus states, building notification/toast
  systems, adding scroll-triggered animations, implementing drag-and-drop, or creating
  delightful user experiences.
---

# Interaction Design

Create engaging, intuitive interactions through motion, feedback, and thoughtful state transitions.

## Project Conventions

- **Preferred library:** Framer Motion for React component animations
- **CSS animations:** Use `transform` and `opacity` only for 60fps performance
- **Spring physics:** Prefer spring animations over linear/eased for interactive elements
- **Interruptible:** Users must be able to cancel or override any animation

### Timing Scale

| Duration  | Use Case                                  |
| --------- | ----------------------------------------- |
| 100-150ms | Micro-feedback (hovers, clicks)           |
| 200-300ms | Small transitions (toggles, dropdowns)    |
| 300-500ms | Medium transitions (modals, page changes) |
| 500ms+    | Complex choreographed animations          |

### Accessibility

Always respect `prefers-reduced-motion: reduce`. Set animation duration to 0 when the user prefers reduced motion. This is non-negotiable.

## Reference Files

Read these on demand based on the task:

| File | When to read |
|------|-------------|
| [animation-libraries.md](references/animation-libraries.md) | Choosing or configuring Framer Motion, GSAP, or CSS-based approaches |
| [microinteraction-patterns.md](references/microinteraction-patterns.md) | Building button feedback, toggles, ripples, loading indicators, toasts |
| [scroll-animations.md](references/scroll-animations.md) | Implementing scroll-triggered reveals, parallax, or intersection-based animations |

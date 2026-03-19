---
name: ts-best-practices
description: >
  TypeScript type safety guidelines for writing maximally type-safe code. Apply these patterns
  when writing or reviewing any TypeScript: discriminated unions, type narrowing, type guards,
  exhaustiveness checks, avoiding `as` casts, preferring `unknown` over `any`, and making
  impossible states unrepresentable. Use this skill whenever writing TypeScript code, reviewing
  TypeScript for type safety issues, or when the user mentions type safety, type narrowing,
  discriminated unions, or asks to make types stricter/more explicit.
---

# Type Safety

This project's TypeScript policy. Apply when writing or reviewing TypeScript.

| Rule | Summary |
|------|---------|
| No `as` casts | Every `as` is a potential runtime crash. Validate at boundaries, then cast only if earned. Prefer Zod/Valibot over manual validation. |
| `unknown` over `any` | `any` disables type checking for everything it touches. External data is always `unknown`. |
| Discriminated unions | Model variants with a shared literal discriminant. No optional-field bags. |
| Narrowing hierarchy | Prefer: discriminated union switch > `in` operator > typeof/instanceof > type guard > `as` |
| Type guards | Must actually verify the claim. Name them `isX` or `hasX`. Prefer discriminant narrowing when possible. |
| Exhaustiveness checks | Always add `default: never` arm to switches over discriminated unions. Use an `absurd()` helper to reduce boilerplate. |
| `satisfies` over `as` | When verifying a value matches a type without widening, use `satisfies` to preserve literal types. |
| Impossible states | If a bug requires asking "can this combination happen?" the type is too loose. Tighten it. |

Read [references/patterns.md](references/patterns.md) for code examples of each rule.

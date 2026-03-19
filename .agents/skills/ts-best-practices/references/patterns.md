# Type Safety Patterns

Code examples for each rule in the project's TypeScript policy.

## Never `as` cast

```ts
// BAD
const user = data as User;

// GOOD — validate at the boundary
function parseUser(data: unknown): User {
  if (typeof data !== "object" || data === null) throw new Error("expected object");
  if (!("id" in data) || typeof (data as Record<string, unknown>).id !== "string")
    throw new Error("expected id");
  // ... validate all fields
  return data as User; // OK — earned cast after full validation
}
```

**Refactoring `as` out of existing code:** Determine *why* TypeScript can't infer the type:
- Missing discriminant field — add one, use discriminated union
- Overly wide type (e.g. `Record<string, any>`) — narrow the type definition
- Untyped API boundary — add a type guard or schema parse at the boundary
- Genuinely impossible to express — use a branded type or `satisfies` instead

## `unknown` over `any`

```ts
// BAD
function handle(input: any) { return input.foo.bar; }

// GOOD
function handle(input: unknown) {
  if (typeof input === "object" && input !== null && "foo" in input) {
    // narrowed — compiler verifies access
  }
}
```

When receiving data from external sources (API responses, JSON parse, event payloads, message passing), always type as `unknown` and narrow.

## Discriminated Unions

```ts
// BAD — optional fields create ambiguous states
type Shape = { kind?: string; radius?: number; width?: number; height?: number };

// GOOD — impossible states are unrepresentable
type Shape =
  | { kind: "circle"; radius: number }
  | { kind: "rect"; width: number; height: number };
```

Rules:
- Discriminant field must be a literal type (string literal, number literal, `true`/`false`)
- Every variant shares the same discriminant field name
- Each variant's discriminant value is unique

## Type Narrowing

```ts
// Narrowing patterns (best to worst):
// 1. Discriminated union switch/if — compiler narrows automatically
// 2. `in` operator — "key" in obj narrows to variants containing that key
// 3. typeof / instanceof — for primitives and class instances
// 4. User-defined type guard — when above aren't sufficient
// 5. `as` cast — last resort, only after validation

// `in` operator narrowing
function area(s: Shape): number {
  if ("radius" in s) return Math.PI * s.radius ** 2; // narrowed to circle
  return s.width * s.height; // narrowed to rect
}
```

## Type Guards

```ts
function isCircle(s: Shape): s is Shape & { kind: "circle" } {
  return s.kind === "circle";
}
```

Rules:
- The guard body must actually verify the claim — a lying guard is worse than `as`
- Prefer discriminated union narrowing over custom guards when possible
- Name guards `isX` or `hasX` for readability

## Exhaustiveness Checks

```ts
function area(s: Shape): number {
  switch (s.kind) {
    case "circle": return Math.PI * s.radius ** 2;
    case "rect": return s.width * s.height;
    default: {
      const _exhaustive: never = s;
      throw new Error(`unhandled shape: ${(_exhaustive as { kind: string }).kind}`);
    }
  }
}
```

Helper to reduce boilerplate:

```ts
function absurd(x: never, msg?: string): never {
  throw new Error(msg ?? `unexpected value: ${JSON.stringify(x)}`);
}

// usage in default arm:
default: return absurd(s, `unhandled shape`);
```

## `satisfies` Over `as`

```ts
// BAD — widens, loses literal types
const config = { theme: "dark", cols: 3 } as Config;

// GOOD — validates AND preserves literal types
const config = { theme: "dark", cols: 3 } satisfies Config;
// config.theme is "dark" (literal), not string
```

## Making Impossible States Unrepresentable

```ts
// BAD — can be { loading: true, data: User, error: Error } simultaneously
type State = { loading: boolean; data?: User; error?: Error };

// GOOD — exactly one state at a time
type State =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "success"; data: User }
  | { status: "error"; error: Error };
```

If a bug requires checking "wait, can this combination actually happen?" — the type is too loose. Tighten it so the type system answers that question at compile time.

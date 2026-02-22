---
name: ast-grep
description: >-
  Write ast-grep rules for structural code search where code shape matters, not
  just text. Use when the query involves relationships between code elements
  that text search cannot express. Examples: "find async functions without error
  handling", "find components that accept children but never render them", "find
  all setState calls not inside useEffect", "refactor foo(a, b) to foo({ a, b
  })". Do NOT use for simple name/string lookups; use Grep for those.
---

# ast-grep

## Workflow

**Use Tasks to track progress.** Create a task for each step below (TaskCreate), mark each in_progress when starting and completed when done (TaskUpdate). Check TaskList after each step.

1. Write a test snippet representing the target code
2. Write the rule (start with `pattern`, escalate to `kind` + `has`/`inside` if needed)
3. Test with `--stdin` before searching the codebase
4. Search the codebase once the rule matches

## Critical Gotchas

### Always use `stopBy: end` on relational rules

Without it, `has`/`inside` stop at the first non-matching node instead of traversing the full subtree:

```yaml
# WRONG — will miss deeply nested matches
has:
  pattern: await $EXPR

# RIGHT
has:
  pattern: await $EXPR
  stopBy: end
```

### Escape metavariables in shell

`$VAR` gets interpreted by the shell. Either escape or single-quote:

```bash
# Double-quoted: escape with backslash
ast-grep scan --inline-rules "id: test
language: javascript
rule:
  pattern: await \$EXPR" .

# Single-quoted: no escaping needed
ast-grep scan --inline-rules 'id: test
language: javascript
rule:
  pattern: await $EXPR' .
```

### Metavariables must be the sole content of an AST node

These **don't work**: `obj.on$EVENT`, `"Hello $WORLD"`, `a $OP b`, `$jq`

Use `$$OP` for unnamed nodes (operators, punctuation). Use `$$$ARGS` for zero-or-more nodes.

## Testing with --stdin

```bash
echo "async function test() { await fetch(); }" | ast-grep scan --inline-rules 'id: test
language: javascript
rule:
  kind: function_declaration
  has:
    pattern: await $EXPR
    stopBy: end' --stdin
```

## Debugging with --debug-query

When rules don't match, inspect the AST to find correct `kind` values:

```bash
ast-grep run --pattern 'your code here' --lang javascript --debug-query=cst
```

Formats: `cst` (all nodes), `ast` (named only), `pattern` (how ast-grep sees your pattern).

## Rule syntax

See `references/rule_reference.md` for the full rule reference (atomic, relational, composite rules, and metavariables).

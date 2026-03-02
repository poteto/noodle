# Orders DSL — Design Sketch

Status: research / pre-plan

## Problem

The current `orders-next.json` format has three pain points:

1. **Flat expressiveness.** Parallel execution is encoded as integer group IDs (`"group": 1`). No way to express conditionals, dependencies between stages, or nested pipelines. The structure is implicit in array ordering.

2. **No composability.** The execute→quality→reflect pipeline is copy-pasted into every order. No reusable patterns, macros, or templates. Every stage repeats `provider`, `model`, `runtime`, `status: "pending"`.

3. **LLM generation friction.** Deep JSON nesting causes bracket-matching errors. Every stage requires 6+ fields even when most are boilerplate. More tokens = more room for syntax errors.

## Constraints

- **LLM-authored only.** The scheduler skill emits it; humans never hand-write it.
- **Go consumer.** The loop parses it into `orderx.OrdersFile` structs. Parser must be implementable in Go stdlib or a small dependency.
- **Crash-safe promotion.** The loop's `consumeOrdersNext` flow (read → merge → write → delete) must still work. Format change is on the wire, not in the internal representation.
- **Backward compatible rollout.** Can support both JSON and the new format during transition (detect by file extension or content sniffing).

## Baseline: Current JSON

A typical 3-stage order:

```json
{
  "orders": [
    {
      "id": "49",
      "title": "implement work orders redesign",
      "plan": ["plans/49-work-orders-redesign/overview"],
      "rationale": "foundation-before-feature: core infra needed by all other work",
      "status": "active",
      "stages": [
        {"task_key": "execute", "skill": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "sprites", "status": "pending"},
        {"task_key": "quality", "skill": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"},
        {"task_key": "reflect", "skill": "reflect", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

**Token count:** ~180 tokens. Parallel groups require adding `"group": N` to each stage and mentally sorting by group number. Two stages with `"group": 1` run concurrently — invisible unless you scan every stage.

## Candidate 1: XML/JSX-like Markup

Tag names are task keys. Nesting expresses structure. Attributes carry configuration.

```xml
<orders>
  <order id="49" title="implement work orders redesign"
         plan="plans/49-work-orders-redesign/overview"
         rationale="foundation-before-feature: core infra needed by all other work">
    <execute provider="codex" runtime="sprites" />
    <quality provider="claude" />
    <reflect provider="claude" />
  </order>
</orders>
```

**Parallel stages** become visually explicit:

```xml
<order id="infra-dual" title="parallel infrastructure setup">
  <parallel>
    <execute provider="codex" runtime="sprites" skill="execute">
      Build the event system
    </execute>
    <execute provider="codex" runtime="sprites" skill="execute">
      Build the type registry
    </execute>
  </parallel>
  <quality provider="claude" />
</order>
```

**With prompt content** as element body (not a JSON string):

```xml
<order id="49" title="implement work orders redesign"
       plan="plans/49-work-orders-redesign/overview"
       rationale="foundation-before-feature">
  <execute provider="codex" runtime="sprites">
    Implement phase 3 of the work orders redesign.
    Focus on the stage lifecycle state machine.
    See brain/plans/49-work-orders-redesign/phase-03.md for details.
  </execute>
  <quality provider="claude" extra-prompt="Previous attempt missed test coverage" />
  <reflect provider="claude" />
</order>
```

**Composable templates** (stretch — would need a macro system):

```xml
<!-- template definition (in skill config, not in orders) -->
<template name="standard-pipeline">
  <execute provider="codex" runtime="sprites" />
  <quality provider="claude" />
  <reflect provider="claude" />
</template>

<!-- usage -->
<order id="49" title="implement work orders redesign">
  <standard-pipeline />
</order>
```

### Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| LLM generation | Mixed | Claude is fluent in XML from training, but research shows XML output has more errors than JSON and costs ~80% more tokens. No constrained decoding for XML. |
| Expressiveness | Strong | `<parallel>` is much clearer than `"group": 1`. Prompts as text content (not escaped JSON strings) are natural. Nesting is explicit. |
| Composability | Strong | Templates/macros via custom elements. `<standard-pipeline />` is readable and expandable. |
| Go parsing | Good | `encoding/xml` is stdlib. Custom tag names need a SAX-style parser or struct-per-tag approach, but workable. |
| Token efficiency | Poor | XML closing tags and verbose attributes. Roughly same or more tokens than JSON. |

## Candidate 2: S-Expressions

Minimal syntax. Parentheses for nesting, keywords for attributes.

```
(orders
  (order :id "49"
         :title "implement work orders redesign"
         :plan "plans/49-work-orders-redesign/overview"
         :rationale "foundation-before-feature"
    (execute :provider codex :runtime sprites)
    (quality :provider claude)
    (reflect :provider claude)))
```

**Parallel stages:**

```
(order :id "infra-dual" :title "parallel infra"
  (parallel
    (execute :provider codex :runtime sprites "Build the event system")
    (execute :provider codex :runtime sprites "Build the type registry"))
  (quality :provider claude))
```

### Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| LLM generation | Weak | Less Lisp in training data. Parenthesis matching is exactly the same problem as JSON bracket matching, possibly worse. |
| Expressiveness | Strong | Nesting is natural. `parallel` as a form works well. |
| Composability | Strong | Macros are idiomatic in s-expr languages. `(deftemplate standard-pipeline ...)` |
| Go parsing | Fair | No stdlib. Simple recursive-descent parser (~200 lines), but it's custom. |
| Token efficiency | Good | Most compact of all candidates. ~100 tokens for the same order. |

## Candidate 3: HCL-like Blocks

Terraform-inspired. Block labels, attributes, nesting.

```hcl
order "49" {
  title     = "implement work orders redesign"
  plan      = ["plans/49-work-orders-redesign/overview"]
  rationale = "foundation-before-feature"

  execute {
    provider = "codex"
    runtime  = "sprites"
  }

  quality {
    provider = "claude"
  }

  reflect {
    provider = "claude"
  }
}
```

**Parallel stages:**

```hcl
order "infra-dual" {
  title = "parallel infra"

  parallel {
    execute {
      provider = "codex"
      runtime  = "sprites"
      prompt   = "Build the event system"
    }
    execute {
      provider = "codex"
      runtime  = "sprites"
      prompt   = "Build the type registry"
    }
  }

  quality {
    provider = "claude"
  }
}
```

### Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| LLM generation | Fair | LLMs see HCL in Terraform configs. Less common than JSON/XML in training data. Brace matching is similar to JSON. |
| Expressiveness | Strong | Blocks with labels are readable. `parallel {}` nesting is clear. |
| Composability | Fair | HCL has modules, but they're path-based. In-file macros would be custom. |
| Go parsing | Good | `hashicorp/hcl` library is mature and Go-native. |
| Token efficiency | Good | More compact than JSON. Roughly comparable to s-expressions. |

## Candidate 4: Enhanced JSON (Keep Format, Fix Problems)

Stay with JSON but address the three pain points through schema changes:

```json
{
  "orders": [
    {
      "id": "49",
      "title": "implement work orders redesign",
      "plan": ["plans/49-work-orders-redesign/overview"],
      "rationale": "foundation-before-feature",
      "stages": [
        {"do": "execute", "with": "codex/sprites"},
        {"do": "quality", "with": "claude"},
        {"do": "reflect", "with": "claude"}
      ]
    }
  ]
}
```

Key changes:
- **`status` dropped from scheduler output.** Loop sets all incoming stages to `pending`. Removes the most-repeated boilerplate field.
- **`do` + `with` shorthand.** `"do": "execute"` replaces `"task_key": "execute", "skill": "execute"`. `"with": "codex/sprites"` replaces `"provider": "codex", "model": "gpt-5.3-codex", "runtime": "sprites"`. The loop expands shorthands using routing defaults.
- **`parallel` arrays** replace group integers:

```json
{
  "id": "infra-dual",
  "title": "parallel infra",
  "stages": [
    {"parallel": [
      {"do": "execute", "with": "codex/sprites", "prompt": "Build the event system"},
      {"do": "execute", "with": "codex/sprites", "prompt": "Build the type registry"}
    ]},
    {"do": "quality", "with": "claude"}
  ]
}
```

- **Templates** as named references that the loop expands:

```json
{
  "id": "49",
  "title": "implement work orders redesign",
  "template": "standard-pipeline",
  "with": "codex/sprites"
}
```

Where `standard-pipeline` is registered in config and expands to execute→quality→reflect.

### Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| LLM generation | Strong | JSON constrained decoding works. LLMs know JSON best. Reduced field count means fewer errors. |
| Expressiveness | Good | `parallel` arrays are explicit. Still JSON-flat for deep nesting though. |
| Composability | Good | Templates via named references. Not as flexible as XML macros but covers the common case. |
| Go parsing | Excellent | `encoding/json` + custom unmarshaler. No new dependencies. |
| Token efficiency | Good | ~60% of current token count. `"do"/"with"` is compact. |

## Candidate 5: Markdown-ish (Wild Card)

What if the scheduler just wrote structured natural language that the loop parsed?

```markdown
## Order: 49 — implement work orders redesign
Plan: plans/49-work-orders-redesign/overview
Rationale: foundation-before-feature

1. execute (codex, sprites)
   Implement phase 3 of the work orders redesign.
2. quality (claude)
3. reflect (claude)
```

**Parallel:**

```markdown
## Order: infra-dual — parallel infra

1. [parallel]
   - execute (codex, sprites): Build the event system
   - execute (codex, sprites): Build the type registry
2. quality (claude)
```

### Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| LLM generation | Strong | LLMs generate markdown extremely well. Most natural output format. Zero syntax anxiety. |
| Expressiveness | Good | Numbered lists = sequential. Nested bullets = parallel. Headings = orders. |
| Composability | Weak | No macro/template mechanism in markdown. |
| Go parsing | Poor | Fragile regex/line parsing. Ambiguity everywhere. One extra newline breaks it. |
| Token efficiency | Excellent | Most compact. Natural language is what LLMs want to produce. |

## Comparison Matrix

| | JSON (current) | XML/JSX | S-expr | HCL | Enhanced JSON | Markdown |
|---|---|---|---|---|---|---|
| LLM generation quality | Good | Mixed | Weak | Fair | **Strong** | **Strong** |
| Expressiveness | Weak | **Strong** | **Strong** | **Strong** | Good | Good |
| Composability | None | **Strong** | **Strong** | Fair | Good | Weak |
| Go parseability | **Excellent** | Good | Fair | Good | **Excellent** | Poor |
| Token efficiency | Poor | Poor | **Good** | **Good** | **Good** | **Excellent** |
| LLM training data coverage | **High** | **High** | Low | Medium | **High** | **High** |

## Recommendation

**Enhanced JSON** is the pragmatic winner — it solves the three pain points without switching formats:

1. **Expressiveness:** `parallel` arrays replace opaque group IDs. Prompt content stays as strings (no escaping changes).
2. **Composability:** Template references cover the 80% case (standard pipeline pattern).
3. **Generation quality:** Stays JSON (best constrained decoding support, highest LLM familiarity), but with ~40% fewer required fields per stage.

**XML/JSX is the ambitious alternative** worth prototyping if Enhanced JSON feels too incremental. The `<parallel>` nesting and template-as-element model is genuinely more expressive. The tradeoff is: no constrained decoding, 80% more tokens, and a custom parser (though `encoding/xml` gets you far). Claude specifically is trained on XML-structured prompts, which may partially offset the general "XML is worse for LLM output" finding.

**S-expressions and HCL** are interesting but the LLM training data gap is real. Markdown is compelling for generation quality but too fragile to parse.

## Open Questions

1. **Two-step generation?** Research suggests best results come from "reason freely, then format." Could the scheduler think in natural language and convert to JSON in a second pass? (Adds latency and cost.)
2. **Constrained decoding for XML?** If Anthropic/OpenAI add XML schema support to structured outputs, the generation quality gap closes significantly.
3. **How bad is the current generation quality actually?** The `.bad` file rename mechanism exists — how often does it trigger in practice? If rarely, the generation quality argument is weaker.
4. **Template registry scope?** Where do templates live — `.noodle.toml`, a separate file, or inlined in the skill?

## Next Steps

- [ ] Audit `.bad` file frequency to quantify current generation errors
- [ ] Prototype Enhanced JSON schema changes (drop `status`, add `do`/`with`/`parallel`)
- [ ] Prototype XML parser as a comparison point
- [ ] Test LLM generation quality: give Claude/Codex the same scheduling scenario and compare output quality across JSON, Enhanced JSON, and XML

# Phase 2: Structured Findings Schema and Reflect Consumption

## Goal

Define a structured findings format that upstream skills (quality, execute, etc.) can include in their output, and update reflect to consume it for precise routing of learnings. Uses a tagged envelope in conversation context for reliable transport.

## Why

The quality → reflect → brain pipeline is the main learning path. Currently quality emits a flat narrative string. Reflect parses this and often misroutes learnings. Structured findings provide typed check results with file paths, evidence, and principle references. This follows [[principles/boundary-discipline]] — structure data at the boundary where it's produced, trust the types downstream.

## Design decisions from adversarial review

**Tagged envelope, not generic JSON scanning.** Round 1 replaced sidecar files with "scan conversation for JSON matching the schema." Round 2 reviewers (all three) flagged this: pasted examples, hypothetical discussions, or stale findings from re-runs would be misrouted. Fix: require a tagged envelope — a `## Structured Findings` markdown heading followed by a fenced JSON block. Reflect only matches this tagged form, and consumes only the **last** such block in the conversation (ignoring earlier blocks from reruns or prior stages).

**No `recurrence` field.** Producers don't have cross-session history. Reflect determines recurrence by checking incoming findings against existing brain notes.

**No `risk_level` in schema — producers emit raw findings, reflect classifies.** Same boundary principle as the curator gate: policy lives in the consumer, not the producer.

## Changes

### New file: `reflect/references/findings-schema.md`

Define the structured findings format. This is a convention — any skill can include it in its output using the tagged envelope.

**Tagged envelope format:**

```
## Structured Findings
```json
{ ... }
```
```

Reflect matches ONLY this tagged form. Untagged JSON is ignored. When multiple tagged blocks exist in a conversation (from reruns or multiple stages), reflect consumes only the **last** one.

**Schema:**

- `findings[]`: array of individual findings
  - `check`: name of the check that produced this finding (e.g., "scope-check", "principle-compliance", "test-evidence")
  - `severity`: high / medium / low
  - `file_path`: specific file where the issue was found (optional)
  - `line_context`: relevant line range or function name (optional)
  - `principle`: wikilink to violated principle (optional, e.g., "principles/prove-it-works")
  - `evidence`: concrete evidence string (error message, test output, diff excerpt)
  - `suggested_action`: what to do about it
- `checks_run[]`: list of check names that were executed (for completeness tracking)
- `summary`: one-line human-readable summary

### `reflect/SKILL.md`

Update step 2 (Scan the conversation) to add structured findings as a source:

- Scan conversation for tagged structured findings blocks (matching the envelope format above). Ignore untagged JSON. If multiple tagged blocks exist, consume only the last one (latest is authoritative).
- If found, use findings for precise routing:
  - Finding with `principle` ref → check brain/ for existing notes on this pattern. If a matching brain note exists (grep for check name, file path, or principle ref), this is recurring — strengthen the note. If no match, create a targeted new note with evidence.
  - Finding with `file_path` + `check` → check if structural enforcement exists (lint rule, test). Propose adding one if not ([[principles/encode-lessons-in-structure]])
  - Finding with `severity: high` + matching existing brain note → escalate as principle candidate for meditate
- If no tagged structured findings in conversation, fall back to narrative scanning (existing behavior)

## Dependency

Full structured output from quality requires a noodle-side change (quality skill emitting tagged structured findings in its stage_message). This is a separate PR. The brainmaxxing-side work (this phase) lands first — reflect processes structured findings when present, falls back to narrative when not.

## Test Cases

1. Conversation includes tagged `## Structured Findings` block with a principle violation matching an existing brain note → reflect strengthens the existing note
2. Conversation includes tagged findings with a file-specific issue → reflect proposes structural enforcement instead of a brain note
3. Conversation includes untagged JSON that happens to match the findings schema → reflect ignores it (falls back to narrative scanning)
4. Conversation includes no structured findings → reflect falls back to narrative scanning, same behavior as today
5. User pastes example findings JSON in discussion → reflect ignores it because it lacks the tagged envelope

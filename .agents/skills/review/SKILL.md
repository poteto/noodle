---
name: review
description: >-
  Review plans and code changes. Walks Architecture → Code Quality → Tests → Performance,
  numbering issues with tradeoff options. Triggers: "review", "review this", "code review",
  "look over this", "check my changes", "audit", "critique", "what do you think", "sanity check".
---

# Review

Thorough, interactive review grounded in project principles. **Do NOT make changes — the review is the deliverable.**

## Autonomous Session Mode

When this skill runs in a non-interactive Noodle execution session (for example `Cook`, `Oops`, or `Repair`):

- **Do not use AskUserQuestion.** For each issue found, state the recommended action directly instead of presenting options.
- **Write findings to a report file** at `brain/audits/review-<date>-<subject>.md` in addition to session output.
- **For high-severity issues**, file a todo in `brain/todos.md` using the `/todo` skill.
- **For low-severity issues**, note them in the report but don't file separate todos.
- Conclude by committing the review report. Do not wait for direction.

**Use Tasks to track progress.** Create a task for each step below (TaskCreate), mark each in_progress when starting and completed when done (TaskUpdate). Check TaskList after each step.

## Step 1 — Load Principles

Read `brain/principles.md`. Follow every `[[wikilink]]` and read each linked principle file. These principles govern review judgments — refer back to them when evaluating issues.

**Do NOT skip this. Do NOT use memorized principle content — always read fresh.**

## Step 2 — Determine Scope

Infer what to review from context — the user's message, recent diffs, or referenced plans/PRs. If genuinely ambiguous (nothing to infer), ask.

Auto-detect review mode from change size:
- **BIG CHANGE** (50+ lines changed, 3+ files, or new architecture) — all sections, at most 4 top issues per section
- **SMALL CHANGE** (under those thresholds) — one issue per section

## Step 3 — Gather Context

For **SMALL CHANGE** reviews, read files directly in the main context — delegation overhead exceeds the cost of reading a few files. For **BIG CHANGE** reviews, delegate exploration to subagents via the `Task` tool.

Spawn exploration agents (subagent_type: `Explore`) to:
- Read the code or plan under review
- Identify dependencies, callers, and downstream effects
- Map relevant types, tests, and infrastructure

Run multiple agents in parallel when investigating independent areas.

## Step 4 — Gather Domain Skills

Check installed skills (`.claude/skills/`) for any that match the review's domain. Common matches:

| Domain | Skill | When |
|--------|-------|------|
| Frontend UI | `frontend-design` | Web UI components, layouts, visual design |
| Codex delegation | `codex` | Tasks delegated to Codex workers |

For domains **not listed above**, use `find-skills` to search for a relevant skill.

**Invoke matched skills now** — read their output and use domain guidance to inform your review.

## Step 5 — Review Sections

Work through all sections, then present the full review. The user can redirect mid-stream if needed.

### 1. Architecture
- System design and component boundaries
- Dependency graph and coupling
- Data flow patterns and bottlenecks
- Scaling characteristics and single points of failure
- Security architecture (auth, data access, API boundaries)

### 2. Code Quality
- Code organization and module structure
- DRY violations — be aggressive
- Error handling patterns and missing edge cases (call out explicitly)
- Technical debt hotspots
- Over-engineering or under-engineering relative to foundational-thinking principles; consider redesign-from-first-principles

### 3. Tests
- Coverage gaps (unit, integration, e2e)
- Test quality and assertion strength
- Missing edge case coverage — be thorough
- Untested failure modes and error paths

### 4. Performance
- N+1 queries and database access patterns
- Memory-usage concerns
- Caching opportunities
- Slow or high-complexity code paths

## Step 6 — Issue Format

**NUMBER** each issue (1, 2, 3...). For every issue:

- Describe the problem concretely with file and line references
- Present 2–3 options with **LETTERS** (A, B, C), including "do nothing" where reasonable
- For each option: implementation effort, risk, impact on other code, maintenance burden
- Give a recommended option and why, mapped to foundational-thinking principles
- Ask whether the user agrees or wants a different direction

When using `AskUserQuestion`, label each option with issue NUMBER and option LETTER. Recommended option is always first.

## Interaction Rules

- Do not assume priorities on timeline or scale
- Do not make changes — present findings and wait for direction
- Present all sections together, then ask for feedback once at the end
- Per prove-it-works: if something can be tested, note how in the issue description

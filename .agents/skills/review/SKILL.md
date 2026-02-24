---
name: review
description: >-
  Review plans and code changes. Walks Scope → Architecture → Code Quality → Tests →
  Performance, numbering issues with tradeoff options. In autonomous sessions, writes
  verdict to .noodle/quality/ and accept/reject decision. Triggers: "review",
  "review this", "code review", or scheduled as post-cook quality gate.
noodle:
  blocking: false
  schedule: "After each cook session completes"
---

# Review

Thorough review grounded in project principles. **Do NOT make changes — the review is the deliverable.**

**Use Tasks to track progress.** Create a task for each step below (TaskCreate), mark each in_progress when starting and completed when done (TaskUpdate).

## Autonomous Session Mode

When running in a non-interactive Noodle session (Cook, Oops, Repair):

- **Never ask the user.** State recommended actions directly.
- **Write verdict** to `.noodle/quality/<session-id>.json` per [references/verdict-schema.md](references/verdict-schema.md).
- **Write report** to `brain/audits/review-<date>-<subject>.md`.
- **High-severity issues**: file a todo in `brain/todos.md` via `/todo`.
- **Low-severity issues**: note in report only.
- **Accept**: all checks pass, scope clean, tests present and passing.
- **Reject**: include specific actionable feedback (file, line context, principle violated). The retry cook must be able to fix issues from feedback alone.
- Commit the report and verdict. Do not wait for direction.

## Step 1 — Load Principles

Read `brain/principles.md`. Follow every `[[wikilink]]` and read each linked file. These govern review judgments.

**Do NOT skip this. Do NOT use memorized principle content — always read fresh.**

## Step 2 — Determine Scope

Infer what to review from context — the user's message, recent diffs, or referenced plans/PRs. If genuinely ambiguous (nothing to infer), ask.

Auto-detect review mode from change size:
- **BIG CHANGE** (50+ lines, 3+ files, or new architecture) — all sections, at most 4 top issues per section
- **SMALL CHANGE** (under those thresholds) — one issue per section

## Step 3 — Gather Context

**SMALL CHANGE**: read files directly — delegation overhead exceeds the cost.
**BIG CHANGE**: spawn exploration agents (subagent_type: `Explore`) to read code, identify dependencies/callers/downstream effects, and map relevant types and tests. Run multiple agents in parallel for independent areas.

## Step 4 — Gather Domain Skills

Check installed skills (`.claude/skills/`) for domain matches:

| Domain | Skill | When |
|--------|-------|------|
| Bubble Tea TUI | `bubbletea-tui` | Terminal UI components, views, styling |
| Go code | `go-best-practices` | Go patterns, concurrency, testing |
| Codex delegation | `codex` | Tasks delegated to Codex workers |

For unlisted domains, use `find-skills` to search.

## Step 5 — Review Sections

Work through all sections, then present findings together.

### 1. Scope
- Read the plan phase the work was assigned (from initial prompt or mise context).
- Run `git diff --stat` and `git log --oneline` for the relevant commits.
- Flag files changed outside the plan phase's stated scope as scope violations.
- Stop early and reject on any high-severity scope violation.

### 2. Architecture
- System design and component boundaries
- Dependency graph and coupling
- Data flow patterns and bottlenecks
- Security architecture (auth, data access, API boundaries)

### 3. Code Quality
- `go vet ./...` and `go test ./...` must pass
- Run `sh scripts/lint-arch.sh` if it exists
- Error handling follows conventions (failure-state messages, not expectation messages)
- DRY violations — be aggressive
- Over-engineering or under-engineering; consider redesign-from-first-principles
- Common principle violations: bolted-on changes (redesign-from-first-principles), missing verification (prove-it-works), unnecessary complexity (subtract-before-you-add)

### 4. Tests
- New behavior must have new tests
- Tests assert outcomes, not implementation details
- Coverage of edge cases mentioned in plan phase
- Untested failure modes and error paths

### 5. Performance
- N+1 queries and database access patterns
- Memory-usage concerns
- Caching opportunities
- Slow or high-complexity code paths

### 6. Runtime Verification
- If the plan phase specifies runtime checks, verify they were performed
- "It compiles" is not verification

## Step 6 — Issue Format

**NUMBER** each issue (1, 2, 3...). For every issue:

- Describe the problem concretely with file and line references
- Present 2–3 options with **LETTERS** (A, B, C), including "do nothing" where reasonable
- For each option: implementation effort, risk, impact, maintenance burden
- Recommended option first, mapped to principles
- In interactive mode: ask whether the user agrees or wants a different direction

When using `AskUserQuestion`, label each option with issue NUMBER and option LETTER.

## Non-blocking Issues

Issues that don't warrant rejection (style nits, minor improvements): file as backlog items via `/todo`. Record their IDs in `todos_created` in the verdict.

## Interaction Rules

- Do not assume priorities on timeline or scale
- Do not make changes — present findings and wait for direction (interactive) or write verdict (autonomous)
- Present all sections together, then ask for feedback once at the end
- Per prove-it-works: if something can be tested, note how in the issue description

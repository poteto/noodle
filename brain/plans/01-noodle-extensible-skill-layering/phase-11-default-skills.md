Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 11 — Default Skills + Adapters

## Goal

Write the default skills and adapter sync scripts that ship in the Noodle repository. These are not embedded in the Go binary — they live in the repo and get installed to `~/.noodle/skills/` by the bootstrap skill. Users override them by placing a skill with the same name in their project's `skills/` directory (committed to git).

Use the `skill-creator` skill when writing each skill.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Skills should be clear, concise SKILL.md files — not sprawling reference manuals. Adapter scripts should be minimal POSIX shell.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **Tests should increase confidence, not coverage.** Test the critical adapter flows (round-trip parse/serialize, tag extraction, ID management, plan phase status derivation). Skip testing trivial formatting or obvious string operations. Ask: "does this test help us iterate faster?"
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.
- **Runtime verification in a fake project is required.** After static checks pass, run Noodle in a fresh temporary git repo under `/tmp` and verify the phase behavior there before closing.


- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

### Skills (in repo, installed to `~/.noodle/skills/` by bootstrap)

- **`sous-chef`** — Reads the mise brief, applies scheduling judgment, outputs a prioritized JSON queue with routing annotations. The skill teaches the agent what the queue format is, how to reason about dependencies and parallelism, and what constraints to respect.
- **`taster`** — Quality reviewer. Checks a cook's work before it's considered done. Reviews diffs, test results, and verification output. Outputs accept/reject with rationale.
- **`commit`** — Commit and merge workflow guidance for autonomous shipping. Teaches cooks to create logical commits with meaningful bodies, then merge verified worktree changes back to `main`.
- **`backlog`** — Teaches agents how to read and write the user's backlog system. The default teaches `brain/todos.md` format. Users with Linear/GitHub Issues/etc. override this with their own backlog skill.
- **`plans`** — Teaches agents how to read and write the user's plan system. The default teaches `brain/plans/` format.
- **`noodle`** — Meta-skill that teaches agents how to operate and configure Noodle. This is the skill loaded by repair sessions and is critical for self-healing. It covers:
  - The `noodle.toml` config schema — every field, what it does, valid values, defaults
  - The directory layout — `noodle.toml` (committed config), `.noodle/` (runtime state), `~/.noodle/` (global)
  - The kitchen brigade model — what each session type does, when it triggers, how they interact
  - The adapter pattern — how to write adapter scripts, the normalized schemas (`BacklogItem`, `PlanItem`), input/output contracts per action
  - Skill resolution — how skill paths work, project vs user precedence, how to override defaults
  - Common repair tasks — fixing missing adapter scripts, correcting stale paths, resolving config issues
  - The `references/` directory should include the config schema, adapter contracts from Phase 8, and normalized schemas as reference docs the agent can consult
- **`reflect`** — Session reflection. Reviews what happened in a cook session — mistakes, corrections, new knowledge — and persists learnings to the brain. Runs at the end of significant sessions.
- **`meditate`** — Brain maintenance. Audits the brain vault for outdated information, discovers connections between learnings to form higher-level principles, and prunes stale content. Runs periodically or on demand.
- **`oops`** — Diagnoses and fixes project infrastructure issues: broken tests, failing builds, workflow problems. This is the default; users with project-specific tooling should override via `phases.oops` in config. Unlike repair (which fixes Noodle itself), oops addresses the user's codebase — and the user won't have Noodle's source code in their project.
- **`debugging`** — Systematic debugging methodology: root-cause analysis, bisection, hypothesis testing. Loaded by oops and repair sessions as a foundation skill. Users can override via `phases.debugging` in config to provide project-specific debugging guidance (custom test commands, known failure modes, environment quirks).
- **`doctor`** — Troubleshooting guide for when Noodle gets into a weird state. Teaches the user's agent how to diagnose and fix common problems: clearing stale `.noodle/` state files, resetting orphaned tmux sessions, regenerating `tickets.json`, fixing corrupted `queue.json` or `mise.json`, rebuilding adapter scripts. The skill walks through diagnosis first (what's the symptom, what state files look wrong) before prescribing fixes. If the issue can't be resolved — a bug in the Go binary, unexpected panics, reproducible crashes — the skill tells the agent to file an issue on the Noodle GitHub repo with the relevant diagnostics (Go version, OS, error output, sanitized state files).
### Default Source Formats

These are the formats the default adapter scripts parse and write. Users with different backlog/plan systems replace the adapter scripts entirely — these specs only apply to the defaults that ship with Noodle.

#### `brain/todos.md`

```markdown
# Todos

<!-- next-id: 6 -->

## Frontend

1. [ ] Fix login bug — auth token refresh race condition #auth #urgent
2. [ ] Add pagination to user list [[plans/02-pagination/overview]]
3. [x] Update favicon

## Backend

4. [ ] Refactor middleware — extract auth into shared package #auth
5. [ ] Add rate limiting to API endpoints #infra ~large

---

Optional preamble above the separator is ignored by the parser.
```

**Format rules:**

| Element | Format | Notes |
|---------|--------|-------|
| Section header | `## Name` | Groups items. Becomes `section` in BacklogItem. |
| Tags | `#tagname` | Inline in description. Extracted into `tags` array. Stripped from `title`. |
| Estimate | `~small`, `~medium`, `~large` | Inline in description. Extracted into `estimate`. Stripped from `title`. |
| Item | `N. [ ] description` | `N` is a unique integer ID. `[ ]` checkbox marks open. |
| Done item | `N. [x] description` | `[x]` checkbox marks completion. Obsidian renders natively. |
| Plan reference | `[[plans/NN-name/overview]]` | Wikilink anywhere in description. Becomes `plan` in BacklogItem. |
| Next-ID marker | `<!-- next-id: N -->` | HTML comment. Tracks next available ID. Incremented on add. |
| Sub-bullets | 4-space indent + `- text` | Continuation lines under an item. Appended to description. |
| Separator | `---` | Optional. Content above is preamble (ignored by parser). |

**ID rules:**
- IDs are positive integers, unique across the entire file (not per-section).
- IDs are never reused — `next-id` only increases.
- The `backlog-add` script reads `next-id`, uses it, and increments.

**Parsing priority:** Items are ordered by their position in the file (top = highest priority within a section). The `backlog-sync` script outputs items in file order — the first item in NDJSON output is highest priority. There is no `priority` field in the `BacklogItem` schema; position in the NDJSON stream *is* the priority. The sous chef reads items in order and uses position as a signal alongside tags, estimates, and plan context.

**Round-trip safety:** The parser preserves all original lines. Add/done/edit operations modify specific lines and rewrite the file. Blank lines, comments, and unrecognized content are preserved.

#### `brain/plans/`

```
brain/plans/
├── index.md                              # list of all plans
└── 01-noodle-extensible-skill-layering/
    ├── overview.md                        # frontmatter + plan description
    ├── phase-01-scaffold.md
    ├── phase-02-config.md
    └── testing.md                         # non-phase files allowed
```

**`index.md`:**

```markdown
# Plans

- [ ] [[plans/01-noodle-extensible-skill-layering/overview]]
- [x] [[plans/02-mobile-app/overview]]
```

A checkbox list of wikilinks to plan overviews. `[ ]` = active/draft, `[x]` = done. Managed by `plan-create` and `plan-done` scripts. Obsidian renders the checkboxes natively.

**`overview.md` frontmatter:**

```yaml
---
id: 1
created: 2026-02-20
updated: 2026-02-22
status: active
---
```

| Field | Type | Required | Values |
|-------|------|----------|--------|
| `id` | int | yes | Matches the todo item ID this plan is linked to |
| `created` | date | yes | `YYYY-MM-DD` |
| `updated` | date | no | `YYYY-MM-DD`, set on any modification |
| `status` | string | yes | `draft`, `active`, `done` |

**Phase files:**

```markdown
Back to [[plans/NN-name/overview]]

# Phase N — Phase Name

## Goal
## Changes
## Data Structures
## Verification
### Static
### Runtime
```

- File naming: `phase-NN-name.md` (zero-padded two-digit number, e.g., `phase-01-scaffold.md`)
- No frontmatter on phase files.
- Back-link to overview at top.
- Phase status is derived from the plan overview's phase list using checkbox syntax, not from phase file contents:

```markdown
## Phases

- [x] [[plans/03-auth-refactor/phase-01-scaffold]] — Scaffold
- [ ] [[plans/03-auth-refactor/phase-02-implement]] — Implement
- [ ] [[plans/03-auth-refactor/phase-03-verify]] — Verify
```

Parse rules:
  - `[x]` → `"done"`
  - `[ ]` → `"pending"`, except the **first** `[ ]` after all `[x]` entries → `"active"`
  - If no checkboxes (plain numbered/bulleted list), all phases are `"pending"` and the first is `"active"`
  - If all phases are `[x]`, plan status should be `"done"`

**Directory naming:** `NN-slug` where `NN` is the zero-padded plan ID and `slug` is a hyphenated lowercase name (e.g., `01-noodle-extensible-skill-layering`).

**plans-sync output:** The `plans-sync` script reads `index.md`, parses each plan's `overview.md` frontmatter, scans the overview body for the checkbox-annotated phases list (see format above), and produces `PlanItem` NDJSON. Phase status is derived entirely from the checkbox state in the overview — the script does not parse phase file contents.

### Default adapter scripts (installed to `.noodle/adapters/` by bootstrap)

Backlog adapter (for `brain/todos.md`):
- **`backlog-sync`** — Parses `brain/todos.md` → `BacklogItem` NDJSON
- **`backlog-add`** — Reads JSON from stdin, appends item to `brain/todos.md`
- **`backlog-done`** — Takes item ID as argument, marks done in `brain/todos.md`
- **`backlog-edit`** — Takes item ID as argument, reads JSON from stdin, updates in `brain/todos.md`

Plans adapter (for `brain/plans/`):
- **`plans-sync`** — Parses `brain/plans/` → `PlanItem` NDJSON
- **`plan-create`** — Reads JSON from stdin, scaffolds plan directory
- **`plan-done`** — Takes plan ID as argument, marks plan complete
- **`plan-phase-add`** — Takes plan ID as argument, reads JSON from stdin, adds phase file

The default adapter scripts that ship with Noodle must be POSIX-compatible (no bash 4+ features) so they work on macOS, Linux, and WSL. Scripts are executables with no required extension. User-written adapter scripts can be anything invocable via `sh -c` — Python, Go binaries, or any other executable. All script execution goes through `sh -c` (see Phase 8 invocation contract).

## Data Structures

No Go types. Skills are SKILL.md documents. Adapter scripts conform to the input/output contracts defined in Phase 8.

## Verification

### Static
- Each skill has a valid SKILL.md
- Adapter scripts pass `shellcheck` (POSIX mode)

### Runtime
- `backlog-sync` on a real `brain/todos.md` outputs valid `BacklogItem` NDJSON
- `backlog-sync` preserves section grouping: items under `## Bugs` have `section: "Bugs"`
- `backlog-sync` detects done items (`[x]` checkbox) and sets `status: "done"`
- `backlog-sync` extracts plan references from wikilinks
- `backlog-sync` assigns priority by position in file
- `backlog-add` with `{title, section}` appends item, increments `next-id`
- `backlog-done 1` toggles item 1's checkbox to `[x]` in `brain/todos.md`
- `backlog-edit 1` with `{title}` replaces item 1's description, preserves sub-bullets
- `backlog-done` + `backlog-sync` round-trip: done item has `status: "done"` in NDJSON
- `plans-sync` on a real `brain/plans/` outputs valid `PlanItem` NDJSON
- `plans-sync` reads frontmatter: id, created, updated, status
- `plans-sync` lists phases with status from overview body
- `plan-create` with `{title, slug}` scaffolds directory, overview with frontmatter, updates index
- `plan-phase-add` creates zero-padded phase file with back-link and template sections
- `plan-done` toggles plan's checkbox to `[x]` in index, marks linked todo as `[x]`
- `noodle skills list` includes `commit` in default installed skills
- A project-level skill with the same name as a default one overrides it in `noodle skills list`

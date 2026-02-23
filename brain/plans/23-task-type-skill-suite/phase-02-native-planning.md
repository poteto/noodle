Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 2: Native Planning — Remove Plan Adapter

## Goal

Replace the plan adapter with native Go code. Plans are read directly from `brain/plans/`, plan mutations are native CLI commands, and the mise brief includes plan metadata without calling an adapter sync script. Quality verdict files are also ingested into the brief.

## Current State

- `[adapters.plans]` in `noodle.toml` configures shell scripts: sync, create, done, phase-add
- `.noodle/adapters/main.go` implements plan operations: `plansSync`, `planCreate`, `planDone`, `planPhaseAdd`
- `adapter.Runner.SyncPlans()` calls `runner.Run(ctx, "plans", "sync", RunOptions{})` → parses NDJSON
- `adapter.PlanItem` struct: ID, Title, Status, Phases, Tags
- `adapter.PlanPhase` struct: Name, Status
- `mise/builder.go:Build()` calls `SyncPlans()` to populate `brief.Plans`
- Plan mutation commands are only called from skills (plan skill uses `go -C noodle run . plan create/done/phase-add`)

## Changes

### Plan reader — create `plan/reader.go`

```go
package plan

// PlanMeta is the parsed YAML frontmatter from a plan's overview.md.
type PlanMeta struct {
    ID      int    `yaml:"id" json:"id"`
    Created string `yaml:"created" json:"created"`
    Status  string `yaml:"status" json:"status"` // draft | active | done
}

// Plan is the fully resolved metadata for one plan directory.
type Plan struct {
    Meta      PlanMeta `json:"meta"`
    Title     string   `json:"title"`
    Directory string   `json:"directory"` // absolute path
    Slug      string   `json:"slug"`      // directory basename (e.g., "23-task-type-skill-suite")
    Phases    []Phase  `json:"phases"`
}

// Phase is one phase file within a plan directory.
type Phase struct {
    Name     string `json:"name"`     // from heading or filename
    Filename string `json:"filename"` // e.g., "phase-01-scaffold.md"
    Status   string `json:"status"`   // pending | active | done (inferred from checklist or default)
}

// ReadAll discovers plans from brain/plans/index.md wikilinks
// and reads metadata from each plan's overview.md frontmatter.
// Silently skips unreadable plans.
func ReadAll(plansDir string) ([]Plan, error)
```

Discovery logic:
1. Read `brain/plans/index.md`
2. Find wikilinks matching `[[plans/NN-slug/overview]]`
3. For each: read `overview.md`, parse YAML frontmatter for `PlanMeta`
4. Extract title from first `# Heading`
5. Discover phase files: list `phase-*.md` files in the plan directory, sorted by number
6. Filter out done plans (status == "done") — same as current adapter behavior

Phase status inference: if the overview has a checklist (`- [x]` / `- [ ]`), use it. Otherwise default all phases to "pending".

Tests (`plan/reader_test.go`):
- Temp `brain/plans/` with index.md, one plan dir with overview.md + phase files
- `ReadAll` returns correct plan count, metadata, title, phases
- Plans with `status: done` are excluded
- Missing index.md returns empty slice (not error)
- Malformed overview.md is skipped silently
- Phase files discovered and sorted by number

### Plan CLI commands — create `plan/commands.go` + `cmd_plan.go`

Move logic from `.noodle/adapters/main.go` (functions `planCreate`, `planDone`, `planPhaseAdd`) into native Go:

```go
package plan

// Create creates a new plan directory with overview.md (with frontmatter) and phase-01.
// Updates brain/plans/index.md with a wikilink.
// Returns the created directory path.
func Create(plansDir string, todoID int, slug string) (string, error)

// Done sets a plan's status to "done" in its overview.md frontmatter.
func Done(plansDir string, planID int) error

// PhaseAdd creates a new numbered phase file in an existing plan directory.
// Auto-numbers based on existing phase files (phase-01, phase-02, etc.).
func PhaseAdd(plansDir string, planID int, phaseName string) (string, error)
```

CLI layer (`cmd_plan.go`):

```go
func runPlanCommand(ctx context.Context, app *App, _ []Command, args []string) error
```

Subcommands:
- `noodle plan create <todo-id> <slug>` — calls `plan.Create()`
- `noodle plan done <plan-id>` — calls `plan.Done()`
- `noodle plan phase-add <plan-id> "Phase Name"` — calls `plan.PhaseAdd()`
- `noodle plan list` — calls `plan.ReadAll()`, prints JSON

Register `plan` in the command catalog alongside existing commands.

Tests:
- `plan/commands_test.go`: `Create` makes dir + overview + phase-01, updates index; `Done` flips status; `PhaseAdd` creates numbered file
- `cmd_plan_test.go`: CLI argument parsing, subcommand routing, error cases

### Mise integration — modify `mise/builder.go` + `mise/types.go`

Replace adapter-based plan sync with native reader:

```go
import "github.com/<module>/plan"

func (b *Builder) Build(ctx context.Context) (Brief, []string, error) {
    // Backlog: still adapter-based (unchanged)
    backlogItems, err := b.runner.SyncBacklog(ctx)
    // ...

    // Plans: native reader (replaces adapter sync)
    plansDir := filepath.Join(b.projectDir, "brain", "plans")
    plans, err := plan.ReadAll(plansDir)
    if err != nil {
        warnings = append(warnings, "plan reading failed: "+err.Error())
    }
    brief.Plans = toPlanSummaries(plans)

    // ... rest unchanged ...
}
```

New types in `mise/types.go`:

```go
// PlanSummary is the plan entry in the mise brief (replaces adapter.PlanItem).
type PlanSummary struct {
    ID        int            `json:"id"`
    Title     string         `json:"title"`
    Status    string         `json:"status"`
    Directory string         `json:"directory"` // slug
    Phases    []PhaseSummary `json:"phases"`
}

type PhaseSummary struct {
    Name     string `json:"name"`
    Filename string `json:"filename"`
    Status   string `json:"status"`
}
```

`Brief.Plans` field changes type from `[]adapter.PlanItem` to `[]PlanSummary`.

### Quality verdict ingestion — create `mise/quality.go`

```go
// QualityVerdict is one quality review result, read from .noodle/quality/.
type QualityVerdict struct {
    SessionID string    `json:"session_id"`
    TargetID  string    `json:"target_id,omitempty"`
    Accept    bool      `json:"accept"`
    Feedback  string    `json:"feedback,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}

// readQualityVerdicts reads verdict files from .noodle/quality/.
// Returns most recent 20, sorted newest first.
func readQualityVerdicts(runtimeDir string) ([]QualityVerdict, error)
```

Reads `*.json` files from `.noodle/quality/` directory. Each file is one verdict.

Add `QualityVerdicts []QualityVerdict` to `Brief`. The builder calls `readQualityVerdicts(b.runtimeDir)` in `Build()`.

**Single-writer contract:** The quality skill agent (Phase 5) is the sole writer of verdict files. It writes to `.noodle/quality/<session-id>.json` as its primary output. Phase 2 only adds the reader infrastructure — the mise builder reads verdicts for the brief, the loop's `runQuality()` spawns the quality session and reads the result. No other code writes verdict files.

Tests (`mise/quality_test.go`):
- Temp `.noodle/quality/` with verdict JSON files
- `readQualityVerdicts` returns correct count, sorted by timestamp
- Non-existent directory returns empty slice
- Malformed JSON files are skipped
- Capped at 20 most recent

### Remove plan adapter — modify `config/config.go` + delete from `adapter/`

**Modify `config/config.go`:**
- Remove `"plans"` entry from `defaultAdapters()` function
- Only `"backlog"` remains in the default adapters map

**Delete from `adapter/types.go`:**
- `PlanItem` struct
- `PlanPhase` struct
- `PlanStatus` type and constants (`PlanStatusDraft`, `PlanStatusActive`, `PlanStatusDone`)
- `PlanPhaseStatus` type and constants

**Delete from `adapter/runner.go`:**
- `SyncPlans()` method

**Delete from `adapter/sync.go`:**
- `ParsePlanItems()` function
- Plan-related validation logic

**Delete from `.noodle/adapters/main.go`:**
- `plansSync`, `planCreate`, `planDone`, `planPhaseAdd` functions
- `plan-*` case branches in the main switch
- Leave backlog functions intact

## Data Structures

- `plan.PlanMeta` — YAML frontmatter (id, created, status)
- `plan.Plan` — full plan (meta, title, slug, directory, phases)
- `plan.Phase` — phase file (name, filename, status)
- `mise.PlanSummary` — plan entry in brief (replaces `adapter.PlanItem`)
- `mise.PhaseSummary` — phase entry in brief (replaces `adapter.PlanPhase`)
- `mise.QualityVerdict` — verdict file (session_id, accept, feedback, timestamp)

## Verification

- `make ci` passes
- `go test ./plan/...` — plan reading and CLI commands
- `go test ./mise/...` — native plan reading in brief, quality verdict ingestion
- `noodle plan create 99 test-plan` creates `brain/plans/99-test-plan/` with overview + phase-01
- `noodle plan done 99` sets status to done in overview frontmatter
- `noodle plan phase-add 99 "Second Phase"` creates `phase-02-second-phase.md`
- Mise brief has `plans` array with native plan data (no adapter)
- Mise brief has `quality_verdicts` array from `.noodle/quality/`
- No `[adapters.plans]` in config schema or defaults
- No `adapter.PlanItem` references remain
- Plan-related functions removed from `.noodle/adapters/main.go`

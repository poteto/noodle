Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 13: Cleanup — Delete Old Skills, Rename, Go Code, Update Stubs

## Goal

Delete the 5 old role-based skills, rename sous-chef→prioritize, update Go task type registry, and ensure `skills/` stubs are consistent with the new `.agents/skills/` versions.

## Changes

### Delete old role-based skills

Remove these directories entirely:
- `.agents/skills/ceo/`
- `.agents/skills/cto/`
- `.agents/skills/director/`
- `.agents/skills/manager/`
- `.agents/skills/operator/`

These have been fully extracted — all valuable patterns are now encoded in the new task-type skills (Phases 1–9).

### Remove verify task type from registry

The verify task type (`internal/taskreg/registry.go`) is no longer needed — the execute skill handles its own verification (tests, lint, plan completeness check). Remove:
- `TaskKeyVerify` constant and its registry entry
- Any verify-related stage transitions in `loop/prioritize.go`
- Any verify references in `config/config.go`

### Rename sous-chef references to prioritize

- Update any references to `sous-chef` in config files, documentation, or Go code
- The task type registry already defaults to skill name `"prioritize"`, so this aligns naming

### Update todos

Mark todo items as done:
- #11 (Remove old role-based skills)
- #12 (Update worktree skill — done in Phase 12)
- #14 (Evaluate interactive skill overlap — addressed by the dual-mode pattern)

## Verification

- Old skill directories are gone: `ls .agents/skills/{ceo,cto,director,manager,operator}` all fail
- No remaining references to `sous-chef` in Go code or config
- Go code compiles: `go build ./...`
- Go tests pass: `go test ./...`
- `go vet ./...` passes
- No `TaskKeyVerify` in `internal/taskreg/registry.go`
- Skill resolver finds all new skills: `go run . skills list`

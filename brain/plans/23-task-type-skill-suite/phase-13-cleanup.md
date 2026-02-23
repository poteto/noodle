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

### Rename sous-chef to prioritize

- Rename `skills/sous-chef/` → `skills/prioritize/`
- The task type registry already defaults to skill name `"prioritize"`, so this aligns them
- Update any references to `sous-chef` in config files or documentation

### Update skills/ stubs (optional — defer if low value)

The `skills/` directory contains Noodle default stubs for users who don't have `.agents/skills/` overrides. Since `.agents/skills/` is the primary extension point, stub updates are lower priority. If done, stubs should:
- Have matching frontmatter (name, description)
- Summarize the skill's purpose in 3–5 lines
- Reference the same contract (input/output format)
- NOT duplicate the full process — that's in `.agents/skills/`

### Update todos

Mark todo items as done:
- #11 (Remove old role-based skills)
- #12 (Update worktree skill — done in Phase 12)
- #14 (Evaluate interactive skill overlap — addressed by the dual-mode pattern)

## Verification

- Old skill directories are gone: `ls .agents/skills/{ceo,cto,director,manager,operator}` all fail
- `skills/sous-chef/` no longer exists → `skills/prioritize/` exists
- Go code compiles: `go build ./...`
- Go tests pass: `go test ./...`
- `go vet ./...` passes
- Skill resolver finds "quality": `go run . skills list`
- All `skills/` stubs have matching names to their `.agents/skills/` counterparts

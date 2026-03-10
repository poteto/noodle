# Codex Temp GOMODCACHE Can Break Go Resolution

- In this Codex environment, `GOMODCACHE` may be injected as `/tmp/codex-go-mod`.
- Failure mode: `go build` / `go test` report `no required module provides package ...` even when the dependency is present in `go.mod` and `go.sum`.
- Diagnostic clue:
  - `go mod download` succeeds.
  - `go list -m <module>` succeeds.
  - `go mod tidy -diff` says the module was found but "does not contain package".
  - The cache zip exists under `/tmp/codex-go-mod/cache/download/...`, but the extracted module directory is an empty read-only shell.
- Workaround: run Go commands with `GOMODCACHE="$HOME/.cache/go-mod"` (or another normal writable persistent path).
- Verified fix:
  - `env GOMODCACHE="$HOME/.cache/go-mod" go build ./...`
  - `noodle worktree exec <name> env GOMODCACHE="$HOME/.cache/go-mod" pnpm check`

See also [[principles/fix-root-causes]], [[principles/prove-it-works]]

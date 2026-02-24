Back to [[plans/27-remote-dispatchers/overview]]

# Phase 5: SpritesBackend implementation

## Status

- Deferred (2026-02-24): SDK-backed `SpritesBackend` not implemented in this cycle.
- Replacement shipped: runtime wrapper approach that executes provider commands via Sprite and reuses tmux dispatcher/session paths.
- Follow-up trigger: implement SDK backend if/when full remote lifecycle ownership (for example richer sync-back) is required.

**Routing:** `codex` / `gpt-5.3-codex` — coding against a clear spec (sprites-go SDK)

## References

- Sprites API: https://sprites.dev/api
- Sprites docs: https://docs.sprites.dev/
- Go SDK: https://github.com/superfly/sprites-go

## Goal

Implement `SpritesBackend` that satisfies the `StreamingBackend` interface using the `sprites-go` SDK. This is the first concrete backend.

## Data structures

- `SpritesBackend` struct — holds sprites client, default sprite name, config
- `spritesHandle` struct — wraps `sprites.Cmd`, sprite name, exec session metadata

## Changes

**`dispatcher/sprites_backend.go` (new)**
Add dependency: `github.com/superfly/sprites-go`

`SpritesBackend.Start`:
1. **Bundle local state:** create a git bundle from the current HEAD via `git bundle create <tmpfile> HEAD`. This captures the current branch including any locally committed changes — no need to push to origin first. The loop already dispatches from dedicated worktrees (cook.go:37) which have clean committed state, so uncommitted changes should not be an issue. If the working tree is somehow dirty, return an error rather than auto-committing — mutating git state with temporary commits adds failure modes.
2. Get or create Sprite: `client.Sprite(name)` (name from config or generated)
3. **Upload bundle to Sprite:** use the Sprites filesystem API to write the bundle file to `/work/repo.bundle` on the VM.
4. **Clone from bundle on the Sprite:** run `git clone /work/repo.bundle /work/repo && cd /work/repo && git remote set-url origin <origin-url>` on the Sprite. Set origin to the real remote URL so the agent can push results back via `git push origin noodle/<session-id>`.
5. Upload prompt file: `sprite.Command("tee", "/work/prompt.txt")` with stdin from prompt content, or use the filesystem API if available
6. Receive the prebuilt command pipeline from the dispatcher via `StreamStartConfig`. The dispatcher owns command assembly (same `buildProviderCommand` logic) — the backend just runs it in the Sprite VM with `/work/repo` as the working directory.
7. Execute: `sprite.CommandContext(ctx, "sh", "-c", pipeline)` with `cmd.StdoutPipe()` for streaming, `cmd.Dir = "/work/repo"`
8. `cmd.Start()` — non-blocking
9. Return `StreamHandle{Stdout: stdoutPipe, ID: sessionInfo, Provider: req.Provider}`

`SpritesBackend.IsAlive`: check if the `sprites.Cmd` process is still running (hasn't returned from `Wait()` yet). Track via a done channel set by a goroutine that calls `cmd.Wait()`.

`SpritesBackend.Kill`: call `cmd.Process.Kill()` or send SIGTERM through the sprites exec kill API.

**`go.mod`**
Add `github.com/superfly/sprites-go` dependency.

## Verification

### Static
- Compiles with sprites-go dependency
- `var _ StreamingBackend = (*SpritesBackend)(nil)` compile-time check

### Runtime
- Boundary test: verify `Start` builds the correct `sprites.Cmd` arguments from a `StreamStartConfig` (mock the sprites client interface, assert command/args/env passed through)
- Boundary test: verify bundle creation captures current HEAD
- Boundary test: verify clone-from-bundle command sets origin to the real remote URL
- Boundary test: verify `IsAlive` tracks process lifecycle via done channel (goroutine calling `cmd.Wait()`)
- Boundary test: verify `Kill` calls the right termination method

No live API tests — the `StreamingDispatcher` tests (Phase 3) cover the full dispatch path with mock backends.

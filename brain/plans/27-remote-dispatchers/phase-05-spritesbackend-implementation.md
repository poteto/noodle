Back to [[plans/27-remote-dispatchers/overview]]

# Phase 5: SpritesBackend implementation

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
1. Get or create Sprite: `client.Sprite(name)` (name from config or generated)
2. Upload prompt file: `sprite.Command("tee", "/work/prompt.txt")` with stdin from prompt content, or use the filesystem API if available
3. Receive the prebuilt command pipeline from the dispatcher via `StreamStartConfig`. The dispatcher owns command assembly (same `buildProviderCommand` logic) — the backend just runs it in the Sprite VM.
4. Execute: `sprite.CommandContext(ctx, "sh", "-c", pipeline)` with `cmd.StdoutPipe()` for streaming
5. `cmd.Start()` — non-blocking
6. Return `StreamHandle{Stdout: stdoutPipe, ID: sessionInfo, Provider: req.Provider}`

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
- Boundary test: verify `IsAlive` tracks process lifecycle via done channel (goroutine calling `cmd.Wait()`)
- Boundary test: verify `Kill` calls the right termination method

No live API tests — the `StreamingDispatcher` tests (Phase 3) cover the full dispatch path with mock backends.

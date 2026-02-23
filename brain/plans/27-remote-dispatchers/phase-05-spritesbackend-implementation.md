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
1. **Pre-dispatch git check:** verify the working tree is clean and the current branch is pushed to origin. Return an error if not — remote dispatch requires committed, pushed code. Check via `git status --porcelain` (must be empty) and `git rev-list HEAD...@{upstream}` (must be empty, meaning local and remote are in sync).
2. Get or create Sprite: `client.Sprite(name)` (name from config or generated)
3. **Clone the repo on the Sprite:** run `git clone --branch <branch> --single-branch --depth=1 <repo-url> /work/repo` on the Sprite. The repo URL comes from the origin remote (`git remote get-url origin`). The branch and commit come from the current HEAD. Check out the exact commit SHA after cloning to guard against races.
4. Upload prompt file: `sprite.Command("tee", "/work/prompt.txt")` with stdin from prompt content, or use the filesystem API if available
5. Receive the prebuilt command pipeline from the dispatcher via `StreamStartConfig`. The dispatcher owns command assembly (same `buildProviderCommand` logic) — the backend just runs it in the Sprite VM with `/work/repo` as the working directory.
6. Execute: `sprite.CommandContext(ctx, "sh", "-c", pipeline)` with `cmd.StdoutPipe()` for streaming, `cmd.Dir = "/work/repo"`
7. `cmd.Start()` — non-blocking
8. Return `StreamHandle{Stdout: stdoutPipe, ID: sessionInfo, Provider: req.Provider}`

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
- Boundary test: verify `Start` returns error when working tree is dirty
- Boundary test: verify `Start` returns error when branch is not pushed to origin
- Boundary test: verify clone command uses correct branch, repo URL, and commit SHA
- Boundary test: verify `IsAlive` tracks process lifecycle via done channel (goroutine calling `cmd.Wait()`)
- Boundary test: verify `Kill` calls the right termination method

No live API tests — the `StreamingDispatcher` tests (Phase 3) cover the full dispatch path with mock backends.

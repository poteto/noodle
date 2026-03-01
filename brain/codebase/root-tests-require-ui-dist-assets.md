# Root Tests Require UI Dist Assets

- Root package tests (`go test ./` or `go test ./...`) compile `embed.go`, which includes `all:ui/dist/client`.
- In worktrees without built UI assets, Go test fails with: `pattern all:ui/dist/client: no matching files found`.
- Use `pnpm build` (which builds UI then Go) or `pnpm --filter noodle-ui build` before running `go test ./...`.
- Backend-only verification can still run with targeted packages (`go test ./loop ./server`) without building UI assets first.
- The `startup` integration test runs `pnpm --filter noodle-ui build` before `go build` to ensure the embed directory exists.

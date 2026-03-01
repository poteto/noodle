# Root Tests Require UI Dist Assets

- Root package tests (`go test ./` or `go test ./...`) compile `embed.go`, which includes `all:ui/dist/client`.
- In worktrees without built UI assets, Go test fails with: `pattern all:ui/dist/client: no matching files found`.
- A temporary local workaround for verification is to create `ui/dist/client/index.html`, run tests, then delete the placeholder file before committing.
- Backend-only verification can still run with targeted packages (`go test ./loop ./server`) without building UI assets first.

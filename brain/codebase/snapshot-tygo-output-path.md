# Snapshot Tygo Output Path

- `go generate ./internal/snapshot` writes generated TS under `internal/snapshot/ui/src/client/` because `tygo` resolves paths from the package working directory.
- The checked-in UI client types live in `ui/src/client/`; when snapshot/session fields change, sync `internal/snapshot/ui/src/client/generated-types.ts` into `ui/src/client/generated-types.ts`.
- Clean up the temporary `internal/snapshot/ui/` tree after syncing so it does not get committed.

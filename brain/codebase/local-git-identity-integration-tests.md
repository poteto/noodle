# Local Git Identity In Integration Tests

`startup/TestCLIIntegrationStartScaffolds` creates a temporary Git repo and runs an empty commit before invoking `noodle start --once`.

CI runners may not have global Git identity configured, so the test must set repository-local identity before commit:

- `git config user.name ...`
- `git config user.email ...`

To avoid environment-level overrides breaking the commit, set commit-scoped env vars too:

- `GIT_AUTHOR_NAME`
- `GIT_AUTHOR_EMAIL`
- `GIT_COMMITTER_NAME`
- `GIT_COMMITTER_EMAIL`

Pattern lives in `startup/integration_test.go` via `configureLocalGitIdentity` and `withEnvOverrides`.

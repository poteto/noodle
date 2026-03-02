# Release Changelog Wrapper Script

- Use `pnpm changelog -- <tag>` to automate release-note bookkeeping.
- Script location: `scripts/release-changelog.mjs`.
- Workflow:
  - Finds the latest prior semver tag merged into `HEAD`.
  - Generates a release section for `previousTag..HEAD` into `CHANGELOG.md` using `conventional-changelog` + `conventional-changelog-conventionalcommits`.
  - Commits `CHANGELOG.md`.
  - Creates an annotated tag for the passed `<tag>` on the changelog commit.

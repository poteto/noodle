# Release Changelog Wrapper Script

- Use `pnpm release -- <tag>` to automate release-note bookkeeping.
- Script location: `scripts/release.mjs`.
- Workflow:
  - Fails fast if the requested tag already exists locally or on `origin`.
  - Finds the latest prior semver tag merged into `HEAD`.
  - If no prior semver tag exists (first release), uses the repository's initial commit as the lower bound.
  - Bumps canonical docs version at `docs/.vitepress/version.ts` and commits it (`docs(version): ...`) before changelog changes.
  - Generates a release section for `previousTag..HEAD` into `CHANGELOG.md` using `conventional-changelog` + `conventional-changelog-conventionalcommits`.
  - Commits `CHANGELOG.md`.
  - Creates an annotated tag for the passed `<tag>` on the changelog commit.

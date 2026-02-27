Back to [[archive/plans/15-bootstrap-onboarding/overview]]

# Phase 2: Homebrew Tap & Binary Distribution

## Goal

Users on macOS can install Noodle with `brew install poteto/tap/noodle`. This is the primary distribution channel for launch.

## Changes

- **New repo: `poteto/homebrew-tap`** — Homebrew tap repository with a `noodle.rb` formula.
- **`.goreleaser.yml`** (new, repo root) — GoReleaser config to build Darwin (amd64 + arm64) binaries and publish to the tap on GitHub release. Limit to macOS for launch; add Linux/Windows targets when those package managers ship.
- **`.github/workflows/release.yml`** (new) — GitHub Actions workflow: on tag push, run GoReleaser, which builds binaries and updates the Homebrew formula.
- **`README.md`** — update Quick Start to show `brew install poteto/tap/noodle`.

## Data Structures

- `noodle.rb` formula: standard Homebrew formula pointing at the GoReleaser-produced tarball.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Cross-repo setup, CI/CD design decisions |

## Verification

### Static
- `.goreleaser.yml` is valid: `goreleaser check`
- Formula syntax is valid Ruby

### Runtime
- Tag a test release, confirm GoReleaser produces binaries
- `brew install poteto/tap/noodle && noodle --help` works on a clean Mac
- Binary version matches the tagged release

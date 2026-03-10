Back to [[plans/114-agent-native-onboarding/overview]]

# Phase 5: README Update

## Goal

Update the README to link to INSTALL.md as the primary onboarding path for agent-assisted setup.

## Changes

**`README.md`:**
- Add a section after "Install" (or replace the sparse "First run" section):
  - "Set up your project" — point your coding agent at `INSTALL.md` and it'll configure everything: skills, config, backlog structure — tailored to your project.
  - Keep the manual path too: "Or follow the [getting started guide](https://noodle-run.github.io/noodle/getting-started) to set up manually."
- The README should make the agent-assisted path the default, with manual as fallback.

## Data Structures

- No code types — markdown only.

## Design Notes

README is the repo's front door. The current README is minimal (good) but doesn't mention INSTALL.md or the agent-assisted flow (gap). This phase closes that gap with 3-4 lines, not a rewrite.

The README should remain short. It's not documentation — it's a signpost. INSTALL.md and the docs site have the details.

## Routing

- Provider: `codex`, Model: `gpt-5.4` (simple markdown edit)

## Verification

### Static
- INSTALL.md link in README resolves to a real file
- No broken markdown links

### Runtime
- Read the README as a new user: is the path to "set up my project" obvious within 10 seconds?

# Directory Fixture Contract

This repository uses directory-native test fixtures.

## Layout

```text
<package>/testdata/
  <fixture-name>/
    expected.md                  # source-of-truth + metadata
    noodle.toml                  # optional fixture-level base config
    state-01/
      input.ndjson               # package-specific state input (optional)
      input.json                 # package-specific state input (optional)
      .noodle/...                # optional runtime snapshot files
      noodle.toml                # optional state override config
    state-02/
    state-03/
```

Rules:

- Fixture names are directory names under `testdata/`.
- State directories must be contiguous and ordered: `state-01`, `state-02`, `state-03`, ...
- `expected.md` is required.
- `schema_version`, `expected_failure`, `bug`, `regression`, and `source_hash` frontmatter keys are required.
- `bug` is a boolean intent flag (`true` means this expected failure is a known bug tracked for future fix; `false` means it is not a tracked bug).
- `regression` is the stable string label for the fixture/regression case.
- `expected.md` frontmatter `schema_version` must match `fixturedir.FixtureSchemaVersion`.
- `source_hash` is derived from fixture input files (all files under the fixture except `expected.md`).
- `noodle fixtures sync` updates `expected.md` `source_hash` values.

## Dev Command

- `noodle fixtures sync`: update `expected.md` `source_hash` values.
- `noodle fixtures check`: fail when any fixture `source_hash` is out of date.

## Config Precedence

Config is resolved with strict precedence:

1. `config.DefaultConfig()`
2. optional fixture-level `noodle.toml`
3. optional state-level `state-XX/noodle.toml`

State-level config always wins over fixture-level config for that state.

## Loop Mapping Rule

Loop fixtures map filesystem state keys directly to expected keys:

- `state-01` directory maps to `expected.states["state-01"]`
- `state-02` directory maps to `expected.states["state-02"]`

No numeric index math is used for expectation lookup.

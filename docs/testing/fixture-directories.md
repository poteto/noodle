# Directory Fixture Contract

This repository uses directory-native test fixtures.

## Layout

```text
<package>/testdata/
  <fixture-name>/
    expected.md
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
- `schema_version`, `expected_failure`, and `bug` frontmatter keys are required.
- `expected.md` frontmatter `schema_version` must match `fixturedir.FixtureSchemaVersion`.

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

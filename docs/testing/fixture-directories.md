# Directory Fixture Contract

This repository uses directory-native test fixtures.

## Layout

```text
<package>/testdata/
  <fixture-name>/
    expected.src.md              # source-of-truth, edited by developers
    expected.md                  # generated from expected.src.md
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
- `expected.src.md` and `expected.md` are both required.
- `schema_version`, `expected_failure`, `bug`, and `regression` frontmatter keys are required.
- `bug` is a boolean intent flag (`true` means this expected failure is a known bug tracked for future fix; `false` means it is not a tracked bug).
- `regression` is the stable string label for the fixture/regression case.
- `expected.md` frontmatter `schema_version` must match `fixturedir.FixtureSchemaVersion`.
- Generated `expected.md` frontmatter includes `source_hash`, derived from `expected.src.md`.
- `expected.md` must be generated from `expected.src.md` via `noodle fixtures sync`.

## Dev Command

- `noodle fixtures sync`: update generated `expected.md` files from `expected.src.md`.
- `noodle fixtures check`: fail when any generated fixture is out of date.

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

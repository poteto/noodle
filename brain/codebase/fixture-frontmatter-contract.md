# Fixture Frontmatter Contract

- `expected.md` is the fixture source of truth.
- `source_hash` in `expected.md` is generated via `make fixtures-hash MODE=sync`.
- `source_hash` must match a deterministic hash of fixture input files (all files under the fixture except `expected.md`).
- Required frontmatter keys in `expected.md`: `schema_version`, `expected_failure`, `bug`, `source_hash`.
- `bug: true` means the expectation captures a known bug that should eventually be fixed (tracked work), not permanent expected behavior.
- `bug: true` requires `expected_failure: true`.

# Fixture Frontmatter Contract

- `expected.src.md` is the editable source of truth.
- `expected.md` is generated via `noodle fixtures sync`; never hand-edit it.
- `source_hash` in generated `expected.md` must match `expected.src.md` content.
- Required frontmatter keys in `expected.src.md`: `schema_version`, `expected_failure`, `bug`, `regression`.
- `bug: true` means the expectation captures a known bug that should eventually be fixed (tracked work), not permanent expected behavior.
- `regression` is the stable string label for identifying the case.
- `bug: true` requires `expected_failure: true`.

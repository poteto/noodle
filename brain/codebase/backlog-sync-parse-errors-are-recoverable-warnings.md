# Backlog Sync Parse Errors Are Recoverable Warnings

- Backlog adapter sync output is NDJSON and can contain mixed valid/invalid lines.
- `adapter.ParseBacklogItems` should skip malformed lines, collect warning strings, and keep valid items.
- Invalid JSON and missing required fields (`id`, `title`) are warning-level and should not fail `mise.Build`.
- Fatal return is reserved for true read failures; parse/validation failures are surfaced through `brief.Warnings`.

See also [[plans/97-adapter-schema-validator/overview]], [[principles/never-block-on-the-human]]

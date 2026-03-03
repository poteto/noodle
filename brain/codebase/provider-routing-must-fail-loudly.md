# Provider Routing Must Fail Loudly

The stamp/parse pipeline previously defaulted unknown NDJSON `type` values to the Claude adapter.
That made unknown provider lines look valid, then disappear when the adapter emitted no canonical events.

Current behavior:

1. `parse.DetectProvider` only accepts known Claude/Codex line types.
2. Unknown or missing `type` now returns an explicit provider-resolution error.
3. `stamp.Processor.ProcessLine` converts routing errors into canonical `error` events instead of dropping lines.

This preserves visibility for routing regressions and prevents silent event loss.

See also [[codebase/claude-code-ndjson-protocol]], [[principles/fix-root-causes]]

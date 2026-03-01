# Runtime Routing Owned By Orders

- Runtime selection belongs to order stages, not skill frontmatter.
- `loop.spawnCook` should send `DispatchRequest.Runtime` from the stage `runtime` field, falling back to `config.Runtime.Default` (default is `"process"`).
- Valid configured runtime names are currently `process`, `sprites`, and `cursor`; `"tmux"` is treated as unknown and emitted as a config warning.
- Skill metadata (`skill.NoodleMeta`) and task registry (`taskreg.TaskType`) should not carry runtime fields.
- Keep non-functional backend tests optional during stub phases; compile-time interface assertions are enough until behavior exists.

See also [[archive/plans/27-remote-dispatchers/overview]], [[principles/boundary-discipline]], [[principles/subtract-before-you-add]]

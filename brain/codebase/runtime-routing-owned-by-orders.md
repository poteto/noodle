# Runtime Routing Owned By Orders

- Runtime selection belongs to order stages, not skill frontmatter.
- `loop.spawnCook` should send `DispatchRequest.Runtime` from the stage `runtime` field with `"tmux"` default.
- Skill metadata (`skill.NoodleMeta`) and task registry (`taskreg.TaskType`) should not carry runtime fields.
- Keep non-functional backend tests optional during stub phases; compile-time interface assertions are enough until behavior exists.

See also [[archive/plans/27-remote-dispatchers/overview]], [[principles/boundary-discipline]], [[principles/subtract-before-you-add]]

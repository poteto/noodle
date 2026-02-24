# Runtime Routing Owned By Queue

- Runtime selection belongs to queue items, not skill frontmatter.
- `loop.spawnCook` should send `DispatchRequest.Runtime` from `queue.runtime` with `"tmux"` default.
- Skill metadata (`skill.NoodleMeta`) and task registry (`taskreg.TaskType`) should not carry runtime fields.
- Keep non-functional backend tests optional during stub phases; compile-time interface assertions are enough until behavior exists.

See also [[plans/27-remote-dispatchers/phase-02-dispatcher-factory-and-runtime-routing]], [[principles/boundary-discipline]]

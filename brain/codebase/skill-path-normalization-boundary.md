# Skill Path Normalization Boundary

- `skills.paths` normalization must happen in one place: `skill.Resolver`.
- Pre-normalizing in callers can break `~` expansion by rewriting paths like `~/.noodle/skills` under the project directory.
- Discovery and dispatch must use the same normalization path; otherwise task-type discovery and runtime dispatch can see different skill sets.
- Treat `skills.paths` as boundary input and hand it to the resolver unchanged.

See also [[principles/boundary-discipline]], [[principles/fix-root-causes]]

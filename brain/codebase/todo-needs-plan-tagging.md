# Todo `#needs_plan` Tagging

When tagging backlog items in `brain/todos.md`, treat a `[[plans/...]]` wikilink as the source of truth that the item already has a plan.

- Do not append `#needs_plan` to open items that already include a plan wikilink.
- If tagging in bulk, use a guard that skips lines containing `[[plans/`.
- Preserve todo IDs and section placement while editing descriptions.

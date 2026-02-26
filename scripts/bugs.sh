#!/bin/sh
# List fixture expected.md files with bug=true in frontmatter.
rg --files -g '**/expected.md' | sort | while IFS= read -r f; do
	if awk 'BEGIN{in_frontmatter=0; found=0}
		/^---[[:space:]]*$/ { in_frontmatter = !in_frontmatter; next }
		in_frontmatter && /^bug:[[:space:]]*true[[:space:]]*$/ { found=1; exit }
		in_frontmatter && /^bug:[[:space:]]*/ { exit }
		END { exit !found }' "$f"; then
		echo "$f"
	fi
done

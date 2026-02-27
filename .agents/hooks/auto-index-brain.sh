#!/bin/bash
# auto-index-brain.sh — PostToolUse hook
# Regenerates brain/index.md when brain/ files are added or removed.
# Emits bare wikilinks — no LLM-generated descriptions.

# Consume hook input
cat > /dev/null

set -euo pipefail

BRAIN_DIR="${CLAUDE_PROJECT_DIR}/brain"
INDEX="${BRAIN_DIR}/index.md"

[ -d "$BRAIN_DIR" ] || exit 0
[ -f "$INDEX" ] || exit 0

# All .md files except index.md — relative paths without .md extension
# Exclude plans/ subdirectories (they have their own index at plans/index.md)
disk=$(find "$BRAIN_DIR" -name "*.md" ! -name "index.md" -type f \
    | sed "s|^${BRAIN_DIR}/||; s|\.md$||" \
    | grep -v "^plans/.\+/" | grep -v "^archive/plans/.\+/" | sort)

# Wikilinks in current index
indexed=$(sed -n 's/.*\[\[\([^]]*\)\]\].*/\1/p' "$INDEX" | sort)

# Exit fast if nothing changed (no new/removed files)
[ "$disk" = "$indexed" ] && exit 0

# --- Drift detected, rebuild ---

# Emit a list of bare wikilinks
emit_files() {
    while IFS= read -r f; do
        [ -z "$f" ] && continue
        echo "- [[$f]]"
    done
}

# Rebuild index
{
    echo "# Brain"
    for section in vision principles delegation codebase; do
        files=$(echo "$disk" | grep "^${section}\(/\|$\)" || true)
        [ -z "$files" ] && continue
        header="$(echo "${section:0:1}" | tr '[:lower:]' '[:upper:]')${section:1}"
        printf '\n## %s\n' "$header"
        echo "$files" | emit_files
    done

    # Standalone sections (not in a subdirectory group)
    for standalone in audits; do
        files=$(echo "$disk" | grep "^${standalone}" || true)
        [ -z "$files" ] && continue
        header="$(echo "${standalone:0:1}" | tr '[:lower:]' '[:upper:]')${standalone:1}"
        printf '\n## %s\n' "$header"
        echo "$files" | emit_files
    done

    # Backlog
    if echo "$disk" | grep -q "^todos$"; then
        printf '\n## Backlog\n'
        echo "- [[todos]]"
    fi

    # Plans are maintained in their own index — just link to it
    if [ -f "${BRAIN_DIR}/plans/index.md" ]; then
        printf '\n## Plans\n'
        echo "- [[plans/index]]"
    fi

    # Archived plans — link to their index if it exists
    if [ -f "${BRAIN_DIR}/archive/plans/index.md" ]; then
        printf '\n## Archived Plans\n'
        echo "- [[archive/plans/index]]"
    fi
    echo ""
} > "$INDEX"

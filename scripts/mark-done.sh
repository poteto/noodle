#!/usr/bin/env sh
set -eu

BRAIN="brain"
TODOS="$BRAIN/todos.md"
ARCHIVE_TODOS="$BRAIN/archive/completed_todos.md"
PLANS_DIR="$BRAIN/plans"
ARCHIVE_PLANS="$BRAIN/archive/plans"

usage() {
  echo "Usage: $0 <todo-id> [--note \"completion note\"]" >&2
  echo "" >&2
  echo "Marks a todo as done: moves its plan to archive and the" >&2
  echo "todo entry to completed_todos.md." >&2
  echo "" >&2
  echo "Example: $0 89 --note \"Deleted NoodleMeta, promoted schedule.\"" >&2
  exit 1
}

if [ "$#" -lt 1 ]; then
  usage
fi

id="$1"
shift

note="done."
while [ "$#" -gt 0 ]; do
  case "$1" in
    --note)
      if [ "$#" -lt 2 ]; then
        echo "--note requires a value" >&2
        exit 1
      fi
      note="done. $2"
      shift 2
      ;;
    --note=*)
      note="done. ${1#--note=}"
      shift
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      ;;
  esac
done

# Verify todos.md exists.
if [ ! -f "$TODOS" ]; then
  echo "not found: $TODOS" >&2
  exit 1
fi

# Extract the todo line.
todo_line=$(grep -n "^${id}\. \[ \]" "$TODOS" || true)
if [ -z "$todo_line" ]; then
  echo "no pending todo with id $id found in $TODOS" >&2
  exit 1
fi
line_num="${todo_line%%:*}"
line_text="${todo_line#*:}"

# Build the completed entry: strikethrough the content, update wikilinks.
# Strip the "N. [ ] " prefix to get the content.
content=$(echo "$line_text" | sed "s/^${id}\. \[ \] //")
# Rewrite plan wikilinks from plans/ to archive/plans/.
content=$(echo "$content" | sed 's|\[\[plans/|\[\[archive/plans/|g')
completed_entry="${id}. [x] ~~${content}~~ — ${note}"

# Prepend to completed_todos.md (after the "# Completed Todos" header + blank line).
if [ ! -f "$ARCHIVE_TODOS" ]; then
  printf "# Completed Todos\n\n%s\n" "$completed_entry" > "$ARCHIVE_TODOS"
else
  # Insert new entry after line 2 (header + blank line).
  tmp=$(mktemp)
  head -2 "$ARCHIVE_TODOS" > "$tmp"
  printf "%s\n" "$completed_entry" >> "$tmp"
  tail -n +3 "$ARCHIVE_TODOS" >> "$tmp"
  mv "$tmp" "$ARCHIVE_TODOS"
fi

# Remove the todo from todos.md.
tmp=$(mktemp)
sed "${line_num}d" "$TODOS" > "$tmp"
mv "$tmp" "$TODOS"

# Remove the id from the priority frontmatter array.
tmp=$(mktemp)
sed "s/${id}, //; s/, ${id}//; s/${id}//" "$TODOS" > "$tmp"
mv "$tmp" "$TODOS"

# Move plan directory to archive if it exists.
plan_dir=$(find "$PLANS_DIR" -maxdepth 1 -type d -name "${id}-*" 2>/dev/null | head -1)
if [ -n "$plan_dir" ]; then
  plan_name=$(basename "$plan_dir")
  mkdir -p "$ARCHIVE_PLANS"
  mv "$plan_dir" "$ARCHIVE_PLANS/$plan_name"
  echo "archived plan: $plan_name"
fi

# Update plans/index.md if it references this plan.
plans_index="$PLANS_DIR/index.md"
if [ -f "$plans_index" ] && grep -q "plans/${id}-" "$plans_index"; then
  tmp=$(mktemp)
  sed "s|- \[ \] \[\[plans/${id}-|- [x] [[archive/plans/${id}-|" "$plans_index" > "$tmp"
  mv "$tmp" "$plans_index"
  echo "updated plans/index.md"
fi

echo "marked todo $id as done"

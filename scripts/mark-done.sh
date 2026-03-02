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

prepend_completed_entry() {
  entry="$1"
  if [ ! -f "$ARCHIVE_TODOS" ]; then
    printf "# Completed Todos\n\n%s\n" "$entry" > "$ARCHIVE_TODOS"
    return
  fi

  tmp=$(mktemp)
  head -2 "$ARCHIVE_TODOS" > "$tmp"
  printf "%s\n" "$entry" >> "$tmp"
  tail -n +3 "$ARCHIVE_TODOS" >> "$tmp"
  mv "$tmp" "$ARCHIVE_TODOS"
}

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

todo_line=""
if [ -f "$TODOS" ]; then
  todo_line=$(grep -n -E "^${id}\. \[[ x]\] " "$TODOS" | head -1 || true)
fi

if [ -n "$todo_line" ]; then
  line_num="${todo_line%%:*}"
  line_text="${todo_line#*:}"

  # Build the completed entry: strikethrough the content, update wikilinks.
  content=$(echo "$line_text" | sed "s/^${id}\. \[[ x]\] //")
  content=$(echo "$content" | sed 's|\[\[plans/|\[\[archive/plans/|g')
  completed_entry="${id}. [x] ~~${content}~~ — ${note}"

  if [ ! -f "$ARCHIVE_TODOS" ] || ! grep -q "^${id}\. \[x\]" "$ARCHIVE_TODOS"; then
    prepend_completed_entry "$completed_entry"
    echo "archived todo: $id"
  fi

  # Remove the todo from todos.md.
  tmp=$(mktemp)
  sed "${line_num}d" "$TODOS" > "$tmp"
  mv "$tmp" "$TODOS"

  # Remove the id from the priority frontmatter array.
  tmp=$(mktemp)
  sed "s/${id}, //; s/, ${id}//; s/${id}//" "$TODOS" > "$tmp"
  mv "$tmp" "$TODOS"
fi

# Normalize stale links for this id in completed todos.
if [ -f "$ARCHIVE_TODOS" ]; then
  tmp=$(mktemp)
  awk -v id="$id" '
    $0 ~ "^" id "\\. \\[x\\] " { gsub("\\[\\[plans/", "[[archive/plans/") }
    { print }
  ' "$ARCHIVE_TODOS" > "$tmp"
  mv "$tmp" "$ARCHIVE_TODOS"
fi

# Move plan directory to archive if it exists.
plan_dir=$(find "$PLANS_DIR" -maxdepth 1 -type d -name "${id}-*" 2>/dev/null | head -1)
if [ -n "$plan_dir" ]; then
  plan_name=$(basename "$plan_dir")
  mkdir -p "$ARCHIVE_PLANS"
  archived_plan_path="$ARCHIVE_PLANS/$plan_name"
  if [ ! -d "$archived_plan_path" ]; then
    mv "$plan_dir" "$archived_plan_path"
    echo "archived plan: $plan_name"
  fi
fi

# Update plans/index.md references for this plan id.
plans_index="$PLANS_DIR/index.md"
if [ -f "$plans_index" ]; then
  tmp=$(mktemp)
  sed \
    -e "s|- \[ \] \[\[plans/${id}-|- [x] [[archive/plans/${id}-|g" \
    -e "s|- \[x\] \[\[plans/${id}-|- [x] [[archive/plans/${id}-|g" \
    "$plans_index" > "$tmp"
  mv "$tmp" "$plans_index"
fi

echo "archived todo/plan for id $id"

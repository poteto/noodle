# Adapter Script Templates

Bootstrap creates scripts in `.noodle/adapters/` and marks them executable.

Scripts must be POSIX shell (`#!/bin/sh`), no bash 4 features.

## Backlog: `brain/todos.md`

### `backlog-sync`

```sh
#!/bin/sh
set -eu

TODOS="brain/todos.md"
[ -f "$TODOS" ] || exit 0

# Minimal fallback: emit open markdown todos as NDJSON.
awk '
  BEGIN { section=""; id=0 }
  /^## / { section=substr($0, 4); next }
  /^[0-9]+\. \[[ xX]\] / {
    line=$0
    status="open"
    if (line ~ /\[[xX]\]/) status="done"
    sub(/^[0-9]+\. \[[ xX]\] /, "", line)
    id++
    printf("{\"id\":\"%d\",\"title\":\"%s\",\"status\":\"%s\",\"section\":\"%s\"}\n", id, escape(line), status, escape(section))
  }
  function escape(s,  t) {
    gsub(/\\/,"\\\\",s)
    gsub(/"/,"\\\"",s)
    gsub(/\r/,"",s)
    gsub(/\t/," ",s)
    return s
  }
' "$TODOS"
```

### `backlog-add`

```sh
#!/bin/sh
set -eu

TODOS="brain/todos.md"
mkdir -p brain
[ -f "$TODOS" ] || {
  cat >"$TODOS" <<'EOF'
# Todos

## Inbox
EOF
}

title="$(cat | sed -n 's/.*"title"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
[ -n "$title" ] || title="Untitled task"

next="$(awk '
  /^[0-9]+\. \[[ xX]\] / {
    split($1, a, ".")
    if (a[1] + 0 > max) max = a[1] + 0
  }
  END { print max + 1 }
' "$TODOS")"

printf "%s. [ ] %s\n" "$next" "$title" >>"$TODOS"
```

### `backlog-done`

```sh
#!/bin/sh
set -eu

id="${1:-}"
[ -n "$id" ] || { echo "usage: backlog-done <id>" >&2; exit 1; }

TODOS="brain/todos.md"
[ -f "$TODOS" ] || exit 0

tmp="$(mktemp)"
awk -v id="$id" '
  {
    if ($0 ~ ("^" id "\\. \\[ \\] ")) {
      sub("^" id "\\. \\[ \\] ", id ". [x] ")
    }
    print
  }
' "$TODOS" >"$tmp"
mv "$tmp" "$TODOS"
```

### `backlog-edit`

```sh
#!/bin/sh
set -eu

id="${1:-}"
[ -n "$id" ] || { echo "usage: backlog-edit <id>" >&2; exit 1; }

TODOS="brain/todos.md"
[ -f "$TODOS" ] || exit 0

title="$(cat | sed -n 's/.*"title"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
[ -n "$title" ] || exit 0

tmp="$(mktemp)"
awk -v id="$id" -v title="$title" '
  {
    if ($0 ~ ("^" id "\\. \\[[ xX]\\] ")) {
      prefix = id ". [ ] "
      if ($0 ~ ("^" id "\\. \\[x\\] ") || $0 ~ ("^" id "\\. \\[X\\] ")) {
        prefix = id ". [x] "
      }
      print prefix title
      next
    }
    print
  }
' "$TODOS" >"$tmp"
mv "$tmp" "$TODOS"
```

## Plans: `brain/plans/`

### `plans-sync`

```sh
#!/bin/sh
set -eu

INDEX="brain/plans/index.md"
[ -f "$INDEX" ] || exit 0

# Minimal stub: emit nothing until plan parser is customized.
exit 0
```

### `plan-create`

```sh
#!/bin/sh
set -eu

mkdir -p brain/plans
[ -f brain/plans/index.md ] || printf "# Plans\n" >brain/plans/index.md
echo "TODO: implement plan-create for your plan format" >&2
```

### `plan-done`

```sh
#!/bin/sh
set -eu

echo "TODO: implement plan-done for your plan format" >&2
```

### `plan-phase-add`

```sh
#!/bin/sh
set -eu

echo "TODO: implement plan-phase-add for your plan format" >&2
```

## External Backlog Templates

For `GitHub Issues`, scaffold `backlog-sync` like:

```sh
#!/bin/sh
set -eu
gh issue list --limit 200 --json number,title,body,state,labels | \
jq -c '.[] | {
  id: (.number|tostring),
  title: .title,
  description: (.body // ""),
  status: (if .state == "OPEN" then "open" else "done" end),
  tags: ((.labels // []) | map(.name))
}'
```

For `Linear` and `Jira`, scaffold executable placeholders with TODO instructions and keep config paths valid.


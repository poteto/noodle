#!/bin/sh

set -u

PRINCIPLES_FILE="brain/principles/boundary-discipline.md"
ERROR_COUNT=0
WARN_COUNT=0

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/lint-arch.XXXXXX")" || exit 1
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

report_violation() {
  level="$1"
  file="$2"
  line="$3"
  violation="$4"
  remediation="$5"

  printf '[ARCH][%s] %s:%s\n' "$level" "$file" "$line"
  printf '  Violation: %s\n' "$violation"
  printf '  Fix: %s\n' "$remediation"
  printf '  See %s\n' "$PRINCIPLES_FILE"
}

report_error() {
  ERROR_COUNT=$((ERROR_COUNT + 1))
  report_violation "ERROR" "$1" "$2" "$3" "$4"
}

report_warn() {
  WARN_COUNT=$((WARN_COUNT + 1))
  report_violation "WARN" "$1" "$2" "$3" "$4"
}

# Build a canonical Go file list from the active codebase.
# Excludes generated/runtime/worktree/legacy directories.
GO_FILES="$TMP_DIR/go-files.txt"
find . \
  \( \
    -path "./.git" -o \
    -path "./.claude/worktrees" -o \
    -path "./.worktrees" -o \
    -path "./.noodle" -o \
    -path "./bin" -o \
    -path "./brain" -o \
    -path "./old_noodle" \
  \) -prune -o \
  -type f -name "*.go" -print | sed 's#^\./##' | sort >"$GO_FILES"

GO_PROD_FILES="$TMP_DIR/go-prod-files.txt"
GO_TEST_FILES="$TMP_DIR/go-test-files.txt"
grep -v '_test\.go$' "$GO_FILES" >"$GO_PROD_FILES" || true
grep '_test\.go$' "$GO_FILES" >"$GO_TEST_FILES" || true

# 1) Go production file size limits.
# Scope: repository production .go files (exclude *_test.go).
# Policy: >1000 lines = ERROR, >500 lines = WARN.
while IFS= read -r file; do
  [ -z "$file" ] && continue
  lines=$(wc -l <"$file" | tr -d ' ')
  if [ "$lines" -gt 1000 ]; then
    report_error \
      "$file" \
      "$lines" \
      "File exceeds 1000 lines ($lines lines)." \
      "Split this file into smaller focused modules."
  elif [ "$lines" -gt 500 ]; then
    report_warn \
      "$file" \
      "$lines" \
      "File exceeds 500 lines ($lines lines)." \
      "Consider splitting by concern to improve readability."
  fi
done <"$GO_PROD_FILES"

# 2) Go test file size limits.
# Scope: repository *_test.go files only.
# Policy: >2000 lines = ERROR, >1200 lines = WARN.
while IFS= read -r file; do
  [ -z "$file" ] && continue
  lines=$(wc -l <"$file" | tr -d ' ')
  if [ "$lines" -gt 2000 ]; then
    report_error \
      "$file" \
      "$lines" \
      "Go test file exceeds 2000 lines ($lines lines)." \
      "Split by feature and extract shared fixtures/helpers to keep test intent readable."
  elif [ "$lines" -gt 1200 ]; then
    report_warn \
      "$file" \
      "$lines" \
      "Go test file exceeds 1200 lines ($lines lines)." \
      "Consider splitting by feature and extracting shared fixtures/helpers."
  fi
done <"$GO_TEST_FILES"

# 3) Legacy fixture files must not exist.
# Fixture contract is directory-based with expected.md.
LEGACY_FIXTURES="$TMP_DIR/legacy-fixture-files.txt"
find . \
  \( \
    -path "./.git" -o \
    -path "./.claude/worktrees" -o \
    -path "./.worktrees" -o \
    -path "./.noodle" -o \
    -path "./bin" \
  \) -prune -o \
  -type f \( -name "*.fixture.md" -o -name "expected.src.md" \) -print | \
  sed 's#^\./##' | sort >"$LEGACY_FIXTURES"

while IFS= read -r file; do
  [ -z "$file" ] && continue
  report_error \
    "$file" \
    "1" \
    "Legacy fixture file detected ($file)." \
    "Use directory fixtures with expected.md and state-XX inputs."
done <"$LEGACY_FIXTURES"

# 4) Boundary failure messages in migrated plan 83 paths must describe failure
# state, not unmet expectations.
MIGRATED_BOUNDARY_FILES="$TMP_DIR/migrated-boundary-files.txt"
cat >"$MIGRATED_BOUNDARY_FILES" <<'EOF'
cmd_start.go
main.go
start_boundary.go
server/server.go
loop/control.go
loop/control_review.go
loop/cook_completion.go
loop/cook_spawn.go
loop/dispatch_failure_envelope.go
loop/failure_envelope.go
loop/failure_projection.go
loop/orders.go
loop/schedule.go
loop/agent_mistake_envelope.go
EOF

EXPECTATION_STYLE_PATTERN='(fmt\.Errorf|errors\.New|http\.Error|newStartAbortEnvelope|newStartRepairPromptEnvelope|newStartWarningOnlyEnvelope|newSystemHardLoopFailure|newOrderHardLoopFailure|newDegradeLoopFailure)[^\n]*"[^"]*\b(must|required|requires|expected)\b[^"]*"'
MATCH_FILE="$TMP_DIR/expectation-style-matches.txt"

while IFS= read -r file; do
  [ -z "$file" ] && continue
  [ -f "$file" ] || continue

  if rg -n --no-heading "$EXPECTATION_STYLE_PATTERN" "$file" >"$MATCH_FILE" 2>/dev/null; then
    while IFS= read -r match; do
      [ -z "$match" ] && continue
      line=$(printf '%s' "$match" | cut -d: -f1)
      report_error \
        "$file" \
        "$line" \
        "Expectation-style wording found in a boundary failure message." \
        "Rewrite the message to describe the observed failure state (for example, '... missing' or '... failed')."
    done <"$MATCH_FILE"
  fi
done <"$MIGRATED_BOUNDARY_FILES"

if [ "$ERROR_COUNT" -gt 0 ]; then
  printf '[ARCH] Failed with %s error(s), %s warning(s).\n' "$ERROR_COUNT" "$WARN_COUNT"
  exit 1
fi

printf '[ARCH] Passed with %s warning(s).\n' "$WARN_COUNT"
exit 0

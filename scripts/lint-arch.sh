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

# 1) Go production file size limits.
# Scope: old_noodle/ production .go files only (exclude *_test.go).
# Policy: >1000 lines = ERROR, >500 lines = WARN.
if [ -d "old_noodle" ]; then
  GO_PROD_FILES="$TMP_DIR/go-prod-files.txt"
  find old_noodle -name '*.go' ! -name '*_test.go' -type f 2>/dev/null | sort >"$GO_PROD_FILES"
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
fi

# 2) Go test file size limits.
# Scope: old_noodle/ *_test.go files only.
# Policy: >2000 lines = ERROR, >1200 lines = WARN.
if [ -d "old_noodle" ]; then
  GO_TEST_FILES="$TMP_DIR/go-test-files.txt"
  find old_noodle -name '*_test.go' -type f 2>/dev/null | sort >"$GO_TEST_FILES"
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
fi

if [ "$ERROR_COUNT" -gt 0 ]; then
  printf '[ARCH] Failed with %s error(s), %s warning(s).\n' "$ERROR_COUNT" "$WARN_COUNT"
  exit 1
fi

printf '[ARCH] Passed with %s warning(s).\n' "$WARN_COUNT"
exit 0

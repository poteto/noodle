#!/bin/sh
# Concatenate all brain .md files into a single snapshot file.
# Usage: snapshot.sh <brain-dir> <output-file>
# Each file is delimited with === path === headers.

brain_dir="${1:?Usage: snapshot.sh <brain-dir> <output-file>}"
output="${2:?Usage: snapshot.sh <brain-dir> <output-file>}"

: > "$output"

find "$brain_dir" -name '*.md' -type f -not -path '*/node_modules/*' -not -path '*/slides/*' | sort | while IFS= read -r f; do
  printf '=== %s ===\n' "$f" >> "$output"
  cat "$f" >> "$output"
  printf '\n\n' >> "$output"
done

echo "$output"

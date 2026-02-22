#!/usr/bin/env sh
set -eu

usage() {
  echo "Usage: $0 <package-dir> <fixture-name> [state-count]" >&2
  echo "Example: $0 loop runtime-repair-sample 2" >&2
  exit 1
}

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
  usage
fi

pkg_dir=$1
fixture_name=$2
state_count=${3:-1}

case "$state_count" in
  ''|*[!0-9]*)
    echo "state-count must be a positive integer" >&2
    exit 1
    ;;
esac

if [ "$state_count" -lt 1 ]; then
  echo "state-count must be >= 1" >&2
  exit 1
fi

fixture_root=$pkg_dir/testdata/$fixture_name
if [ -e "$fixture_root" ]; then
  echo "fixture already exists: $fixture_root" >&2
  exit 1
fi

mkdir -p "$fixture_root"
index=1
while [ "$index" -le "$state_count" ]; do
  state_id=$(printf 'state-%02d' "$index")
  state_dir=$fixture_root/$state_id
  mkdir -p "$state_dir"
  : > "$state_dir/input.ndjson"
  index=$((index + 1))
done

cat > "$fixture_root/expected.md" <<EOF_EXPECTED
---
schema_version: 1
expected_failure: false
bug: false
source_hash: pending
---

## Runtime Dump

\`\`\`json
{}
\`\`\`
EOF_EXPECTED
go run ./scripts/fixturehash sync --root "$fixture_root"

echo "Created fixture: $fixture_root"

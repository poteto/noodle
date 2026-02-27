#!/usr/bin/env sh
set -eu

template="loop"

usage() {
  echo "Usage: $0 [--template loop|snapshot|generic] <package-dir> <fixture-name> [state-count]" >&2
  echo "Example: $0 loop my-new-scenario 2" >&2
  echo "Example: $0 --template snapshot internal/snapshot my-fixture" >&2
  exit 1
}

# Parse --template flag.
while [ "$#" -gt 0 ]; do
  case "$1" in
    --template)
      if [ "$#" -lt 2 ]; then
        echo "--template requires a value (loop|snapshot|generic)" >&2
        exit 1
      fi
      template=$2
      shift 2
      ;;
    --template=*)
      template="${1#--template=}"
      shift
      ;;
    *)
      break
      ;;
  esac
done

case "$template" in
  loop|snapshot|generic) ;;
  *)
    echo "unknown template: $template (expected loop|snapshot|generic)" >&2
    exit 1
    ;;
esac

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
  case "$template" in
    loop)
      : > "$state_dir/input.ndjson"
      ;;
    snapshot)
      echo '{}' > "$state_dir/input.json"
      ;;
    generic)
      # Empty state dir — runner fills content.
      ;;
  esac
  index=$((index + 1))
done

cat > "$fixture_root/expected.md" <<EOF_EXPECTED
---
schema_version: 1
expected_failure: false
bug: false
source_hash: pending
---
EOF_EXPECTED
go run ./scripts/fixturehash sync --root "$fixture_root"

echo "Created fixture ($template): $fixture_root"

#!/bin/bash -e

# Ensures that vnames.json can be read by the //kythe/storage/go/filevnames library

CAMPFIRE_ROOT="$(readlink -e "$PWD")"
DIRECTORY_INDEXER="${CAMPFIRE_ROOT}/campfire-out/bin/third_party/kythe/go/storage/tools/directory_indexer"
CONFIG="${CAMPFIRE_ROOT}/third_party/kythe/data/vnames.json"
OUT="${CAMPFIRE_ROOT}/campfire-out/gen/third_party/kythe/data/file_entries"

# Directory tree with some (but not many) files
DIR="${CAMPFIRE_ROOT}/third_party/kythe/go/platform"

mkdir -p "$(dirname "$OUT")"
cd "$DIR"
"$DIRECTORY_INDEXER" --vnames "$CONFIG" >"$OUT"

test -s "$OUT" || {
  echo "$OUT is empty" >&2
  exit 1
}

#!/bin/bash
#
# Upgrades the third_party/kythe directory to a version in a local Kythe repository.

set -ux

usage() {
  echo "Usage: $(basename "$0") <kythe-repository-root>" >&2
  exit 1
}

case $# in
  1)
    KYTHE_REPO="$(readlink -e "$1")" ;;
  *)
    usage ;;
esac

cd "$(dirname "$0")"/../..

echo "Kythe repository: $KYTHE_REPO" >&2

COMMIT=$(git --git-dir="$KYTHE_REPO/.git" rev-parse HEAD)
echo "Upgrading third_party/kythe to version $COMMIT" >&2

kythe_paths=(
 "proto"
 "go/platform/analysis"
 "go/platform/delimited"
 "go/platform/kindex"
 "java/com/google/devtools/kythe/common"
 "java/com/google/devtools/kythe/extractors/shared"
 "java/com/google/devtools/kythe/platform/shared"
 "java/com/google/devtools/kythe/platform/java"
 "java/com/google/devtools/kythe/platform/java/filemanager"
 )

for path in "${kythe_paths[@]}"; do
  rm -rf third_party/kythe/$path
  mkdir -p third_party/kythe/$path
  cp $KYTHE_REPO/kythe/$path/* third_party/kythe/$path
done

# Fix all the imports
grep -lR '"third_party/kythe/' third_party/kythe | xargs sed -i 's#"third_party/kythe/#"third_party/kythe/#g'
grep -lR //kythe third_party/kythe | grep CAMPFIRE | xargs sed -i 's#//kythe/#//third_party/kythe/#g'

# Update README.google
sed -ri "s/Version: .+/Version: $COMMIT/
s#/tree/.+/kythe/#/tree/$COMMIT/kythe/#" third_party/kythe/README.google

# Clean up CAMPFIRE files
./campfire camper third_party/kythe

# Check that everything still builds
./campfire build third_party/kythe/...
./campfire build ...

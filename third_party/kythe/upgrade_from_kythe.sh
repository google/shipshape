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

rsync --delete --files-from=third_party/kythe/imported_paths_from_kythe $KYTHE_REPO/kythe third_party/kythe

# Fix all the imports
grep -lR '"third_party/kythe/' third_party/kythe | xargs sed -i 's#"third_party/kythe/#"third_party/kythe/#g'
grep -lR //kythe third_party/kythe | grep BUILD | xargs sed -i 's#//kythe/#//third_party/kythe/#g'

# Update README.google
sed -ri "s/Version: .+/Version: $COMMIT/" third_party/kythe/README.google
sed -ri "s#/kythe/tree/.+#/kythe/tree/$COMMIT#" third_party/kythe/README.google

# Clean up
bazel clean

# Check that everything still builds
bazel build //third_party/kythe/...
bazel build ...

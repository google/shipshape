#!/bin/bash -e
#
# Upgrades the third_party/buildtools import to a version in a local Kythe repository.

usage() {
  echo "Usage: $(basename "$0") <kythe-repository-root>" >&2
  exit 1
}

case $# in
  1)
    KYTHE="$(readlink -e "$1")" ;;
  *)
    usage ;;
esac

cd $(dirname "$0")/../..

echo "Kythe repository: $KYTHE" >&2

COMMIT=$(git --git-dir="$KYTHE/.git" rev-parse HEAD)
echo "Upgrading third_party/buildtools to version $COMMIT" >&2
rsync -r --delete "$KYTHE/buildtools/" third_party/buildtools

# Restore Shipshape files
git checkout third_party/buildtools/{LICENSE,README.google,upgrade_from_kythe.sh,buildtools.patch}

# Add Shipshape modifications
rm -rf third_party/buildtools/docker third_party/buildtools/kythe_rules.js
patch -p1 < third_party/buildtools/buildtools.patch

# Update README.google
sed -ri "s/Version: .+/Version: $COMMIT/
s#/tree/.+/buildtools#/tree/$COMMIT/buildtools/#" third_party/buildtools/README.google

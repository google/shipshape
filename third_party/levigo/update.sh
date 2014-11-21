#!/bin/sh -e
#
# This script will update the Kythe build of levigo.a.
# Run without arguments from the root of the Kythe repository.
#
LEVIGO_DIR=$(pwd)/third_party/levigo
LEVELDB_DIR=$(pwd)/third_party/leveldb
set -x
env GOPATH=$LEVIGO_DIR \
    CGO_CFLAGS="-I${LEVELDB_DIR}/include" \
    CGO_LDFLAGS="-L${LEVELDB_DIR}" \
go get github.com/jmhodges/levigo

mv $LEVIGO_DIR/pkg/linux_amd64/github.com/jmhodges/levigo.a $LEVIGO_DIR
rm -fr $LEVIGO_DIR/{src,pkg}
git commit -m "Update levigo.a." -a
set +x

echo "
--- UPDATE COMPLETE ---
"levigo.a has been updated; run 'git push origin HEAD:refs/for/master'
to send the change for review." 2>&1

#!/bin/sh -e
# Packages gson-proto-0.5.jar from the google-gson svn repository.

DIR="$(readlink -e "$(dirname "$0")")"

svn checkout -r1300 http://google-gson.googlecode.com/svn/trunk/ /tmp/google-gson
cd /tmp/google-gson/proto
svn revert -R .
svn patch "$DIR"/gson-proto.patch
mvn package
cp -f target/gson-proto.jar "$DIR"/gson-proto-0.5.jar

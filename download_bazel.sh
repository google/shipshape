#!/bin/bash
#
# Download and install a known-good version of Bazel.
#
# Example:
#   ./download_bazel.sh /tmp/

TARGET=$1

cd $TARGET && \
  git clone https://github.com/google/bazel/ && \
  cd bazel && \
  git checkout 14cd308832a681af9a6755cd01ca145c58a318f6 && \
  ./compile.sh

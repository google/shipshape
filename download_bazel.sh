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
  git checkout e0ac088ebef59ad8d6bf2b315434d7cce627000c && \
  ./compile.sh

#!/bin/bash
#
# Download and install a known-good version of Bazel.
#
# Example:
#   ./download_bazel.sh /tmp/

TARGET=$1

cd $TARGET && \
  git clone -b 0.1.0 --depth 1 https://github.com/google/bazel/ && \
  cd bazel && \
  ./compile.sh

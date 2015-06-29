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
  git checkout 43b2ea7274fe11f0d5c9f3363110da72c37da923 && \
  ./compile.sh

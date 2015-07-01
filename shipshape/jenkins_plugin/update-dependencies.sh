#!/bin/bash

# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script is to be run from the current directory (the root directory of the
# plugin).

# The jenkins plugin depends on jars from kythe which need to be available in
# the local file repository for maven. This repository is typically located in
# ~/.m2/repository.

# Build jars
bazel build //...

# The plugin needs protos from Shipshape:
mvn install:install-file -Dfile=../../bazel-bin/shipshape/proto/libshipshape_rpc_proto_java.jar -DgroupId=com.google.code -DartifactId=shipshape-rpc-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../bazel-bin/shipshape/proto/libshipshape_context_proto_java.jar -DgroupId=com.google.code -DartifactId=shipshape-context-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../bazel-bin/shipshape/proto/libsource_context_proto_java.jar -DgroupId=com.google.code -DartifactId=shipshape-repo-context-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../bazel-bin/shipshape/proto/libnote_proto_java.jar -DgroupId=com.google.code -DartifactId=shipshape-note-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../bazel-bin/shipshape/proto/libtextrange_proto_java.jar -DgroupId=com.google.code -DartifactId=shipshape-textrange-proto -Dversion=1.0 -Dpackaging=jar

# And also some protos from kythe;
mvn install:install-file -Dfile=../../bazel-bin/third_party/kythe/proto/libanalysis_proto_java.jar -DgroupId=com.google.code -DartifactId=kythe-analysis-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../bazel-bin/third_party/kythe/proto/libstorage_proto_java.jar -DgroupId=com.google.code -DartifactId=kythe-storage-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../bazel-bin/third_party/kythe/proto/libany_proto_java.jar -DgroupId=com.google.code -DartifactId=kythe-any-proto -Dversion=1.0 -Dpackaging=jar

# And also jars from third party:
mvn install:install-file -Dfile=../../third_party/gson/gson-2.3-SNAPSHOT.jar -DgroupId=com.google.code -DartifactId=gson -Dversion=2.3 -Dpackaging=jar

# The jars should now be available under ~/.m2/repository/com/google/code/.


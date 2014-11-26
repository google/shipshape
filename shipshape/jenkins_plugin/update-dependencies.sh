#!/bin/bash

# This script is to be run from the current directory (the root directory of the
# plugin).

# The jenkins plugin depends on jars from kythe which need to be available in
# the local file repository for maven. This repository is typically located in
# ~/.m2/repository. To make the kythe jars available in this repo:

# The plugin also needs protos from Shipshape:

mvn install:install-file -Dfile=../../campfire-out/bin/shipshape/proto/shipshape_rpc_proto.jar -DgroupId=com.google.code -DartifactId=shipshape-rpc-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../campfire-out/bin/shipshape/proto/shipshape_context_proto.jar -DgroupId=com.google.code -DartifactId=shipshape-context-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../campfire-out/bin/shipshape/proto/source_context_proto.jar -DgroupId=com.google.code -DartifactId=shipshape-repo-context-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../campfire-out/bin/shipshape/proto/note_proto.jar -DgroupId=com.google.code -DartifactId=shipshape-note-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../campfire-out/bin/shipshape/proto/textrange_proto.jar -DgroupId=com.google.code -DartifactId=shipshape-textrange-proto -Dversion=1.0 -Dpackaging=jar

# And also some protos from kythe;

#mvn install:install-file -Dfile=../../campfire-out/bin/kythe/proto/compilation_proto.jar -DgroupId=com.google.code -DartifactId=kythe-compilation-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../campfire-out/bin/third_party/kythe/proto/analysis_proto.jar -DgroupId=com.google.code -DartifactId=kythe-analysis-proto -Dversion=1.0 -Dpackaging=jar
mvn install:install-file -Dfile=../../campfire-out/bin/third_party/kythe/proto/storage_proto.jar -DgroupId=com.google.code -DartifactId=kythe-storage-proto -Dversion=1.0 -Dpackaging=jar

# And also jars from third party:

mvn install:install-file -Dfile=../../third_party/gson/gson-2.3-SNAPSHOT.jar -DgroupId=com.google.code -DartifactId=gson -Dversion=2.3 -Dpackaging=jar

# The jars should now be available under ~/.m2/repository/com/google/code/.


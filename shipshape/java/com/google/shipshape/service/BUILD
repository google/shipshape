# Copyright 2015 Google Inc. All rights reserved.
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

package(default_visibility = ["//shipshape:default_visibility"])

java_library(
    name = "api",
    javacopts = [
        "-source 8",
        "-target 8",
    ],
    srcs = [
        "Analyzer.java",
        "AnalyzerException.java",
        "AnalyzerStateBuilder.java",
        "JavaDispatcher.java",
        "ShipshapeLogger.java",
        "StatelessAnalyzer.java",
    ],
    deps = [
        "//shipshape/java/com/google/shipshape/util/rpc:protocol",
        "//shipshape/java/com/google/shipshape/util/rpc:server",
        "//shipshape/proto:note_proto_java",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:shipshape_rpc_proto_java",
        "//third_party/guava:guava",
        "//third_party/jsr305_annotations:jsr305",
        "//third_party/kythe/java/com/google/devtools/kythe/common:common",
        "//third_party/kythe/proto:analysis_proto_java",
        "//third_party/openjdk:tools_jar",
        "//third_party/proto:protobuf_java",
    ],
)
java_binary(
    name = "java_dispatcher",
    srcs = [
        "JavaService.java",
    ],
    deps = [
        ":api",
        "//shipshape/java/com/google/shipshape/analyzers:analyzers",
        "//shipshape/java/com/google/shipshape/util/rpc:server",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:shipshape_rpc_proto_java",
        "//third_party/guava:guava",
        "//third_party/jcommander:jcommander",
        "//third_party/jsr305_annotations:jsr305",
        "//third_party/kythe/java/com/google/devtools/kythe/common:common",
        "//third_party/openjdk:tools_jar",
        "//third_party/proto:protobuf_java",
    ],
    main_class = "com.google.shipshape.service.JavaService",
)
java_binary(
    name = "javac_dispatcher",
    srcs = [
        "JavacService.java",
        "JavacStateBuilder.java",
    ],
    deps = [
        ":api",
        "//shipshape/java/com/google/shipshape/analyzers:analyzers",
        "//shipshape/java/com/google/shipshape/util/rpc:server",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:shipshape_rpc_proto_java",
        "//third_party/guava:guava",
        "//third_party/jcommander:jcommander",
        "//third_party/jsr305_annotations:jsr305",
        "//third_party/kythe/java/com/google/devtools/kythe/common:common",
        "//third_party/kythe/java/com/google/devtools/kythe/extractors/shared:shared",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/java:java",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/shared:file_data",
        "//third_party/kythe/proto:analysis_proto_java",
        "//third_party/kythe/proto:storage_proto_java",
        "//third_party/openjdk:tools_jar",
        "//third_party/proto:protobuf_java",
    ],
    main_class = "com.google.shipshape.service.JavacService",
)

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

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

java_library(
    name = "java",
    srcs = [
        "JavaCompilationDetails.java",
        "JavacAnalysisDriver.java",
        "JavacAnalyzer.java",
        "JavacOptionsUtils.java",
    ],
    deps = [
        "//third_party/guava:guava",
        "//third_party/jsr305_annotations:jsr305",
        "//third_party/kythe/java/com/google/devtools/kythe/common:common",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/java/filemanager:filemanager",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/shared:shared",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/shared:file_data",
        "//third_party/kythe/proto:analysis_proto_java",
        "//third_party/kythe/proto:storage_proto_java",
        "//third_party/openjdk:tools_jar",
        "//third_party/proto:protobuf_java",
    ],
)

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
    name = "analyzers",
    srcs = [
        "CheckstyleGoogleAnalyzer.java",
        "CheckstyleUtils.java",
        "EncodingOffsetConverter.java",
        "ErrorProneAnalyzer.java",
        "FixAndReplacementConverter.java",
        "JavacAnalyzer.java",
        "PostMessageAnalyzer.java",
    ],
    deps = [
        "//shipshape/java/com/google/shipshape/service:api",
        "//shipshape/proto:note_proto_java",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:source_context_proto_java",
        "//shipshape/proto:textrange_proto_java",
        "//third_party/error-prone:error-prone",
        "//third_party/guava:guava",
        "//third_party/jsr305_annotations:jsr305",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/java:java",
        "//third_party/kythe/proto:analysis_proto_java",
        "//third_party/kythe/proto:storage_proto_java",
        "//third_party/openjdk:tools_jar",
        "//third_party/proto:protobuf_java",
    ],
)

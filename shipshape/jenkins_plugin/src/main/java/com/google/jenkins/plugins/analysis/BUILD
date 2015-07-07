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
    name = "shipshape",
    srcs = [
        "AnalysisRunner.java",
    ],
    deps = [
        "//shipshape/java/com/google/shipshape/util/rpc:client",
        "//shipshape/proto:note_proto_java",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:shipshape_rpc_proto_java",
        "//shipshape/proto:source_context_proto_java",
        "//shipshape/proto:textrange_proto_java",
        "//third_party/guava:guava",
        "//third_party/jenkins:access-modifier-annotation",
        "//third_party/jenkins:bridge-method-annotation",
        "//third_party/jenkins:bytecode-compatibility-transformer",
        "//third_party/jenkins:jenkins-core",
        "//third_party/jenkins:json-lib",
        "//third_party/jenkins:remoting",
        "//third_party/jenkins:servlet-api",
        "//third_party/jenkins:stapler",
        "//third_party/proto:protobuf_java",
    ],
)

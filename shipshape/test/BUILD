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

load("/tools/build_rules/go", "go_binary")

go_binary(
    name = "test_shipshape_client",
    srcs = [
        "test_shipshape_client.go",
    ],
    deps = [
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/proto:source_context_proto_go",
        "//shipshape/util/file:file",
        "//shipshape/util/rpc/client:client",
        "//third_party/go:protobuf",
    ],
)

java_binary(
    name = "test_request_via_stream",
    srcs = [
        "TestRequestViaStream.java",
    ],
    deps = [
        "//shipshape/proto:note_proto_java",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:shipshape_rpc_proto_java",
        "//shipshape/proto:source_context_proto_java",
        "//third_party/proto:protobuf_java",
    ],
    main_class = "test.TestRequestViaStream",
)

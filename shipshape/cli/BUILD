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

load("/tools/build_rules/go", "go_binary", "go_library", "go_test")

go_binary(
    name = "shipshape",
    srcs = [
        "shipshape.go",
    ],
    deps = [
        ":cli",
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
    ],
)

go_library(
    name = "cli",
    srcs = [
        "shipshape_lib.go",
    ],
    deps = [
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/service:service",
        "//shipshape/util/docker:docker",
        "//shipshape/util/rpc/client:client",
        "//shipshape/util/rpc/server:server",
        "//third_party/go-glog:go-glog",
        "//third_party/go:protobuf",
    ],
)

go_test(
    name = "shipshape_test",
    srcs = [
        "shipshape_test.go",
    ],
    data = [
        "testdata",
    ],
    library = ":cli",
)

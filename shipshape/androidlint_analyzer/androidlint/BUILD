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

load("/tools/build_rules/go", "go_library", "go_binary", "go_test")

go_library(
    name = "androidlint",
    srcs = [
        "analyzer.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:textrange_proto_go",
        "//third_party/go:protobuf",
    ],
)

go_binary(
    name = "android_lint_service",
    srcs = [
        "service.go",
    ],
    deps = [
        ":androidlint",
        "//shipshape/api:api",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/util/rpc/server:server",
        "//third_party/go:protobuf",
    ],
)

go_test(
    name = "androidlint_test",
    srcs = [
        "analyzer_test.go",
    ],
    deps = [
        "//shipshape/util/test:test",
    ],
    data = glob(["testdata/**/*"]) + [
        # This is created by setup_bazel.sh. It is a symlink to the
        # Android Lint binary installed on the system.
        ":lint",
    ],
    library = ":androidlint",
)

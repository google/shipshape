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
        "//shipshape/util/defaults:defaults",
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
        "//shipshape/util/defaults:defaults",
        "//shipshape/util/docker:docker",
        "//shipshape/util/rpc/client:client",
        "//shipshape/util/rpc/server:server",
        "//third_party/go-glog:go-glog",
        "//third_party/go:protobuf",
    ],
)

go_test(
    name = "test_prod",
    srcs = [
        "shipshape_test.go",
    ],
    data = [
        "testdata",
    ],
    library = ":cli",
    deps = [
        "//shipshape/util/docker:docker",
    ],
    args = [
        "--shipshape_test_docker_tag=prod",
    ],
)

go_test(
    name = "test_staging",
    srcs = [
        "shipshape_test.go",
    ],
    data = [
        "testdata",
    ],
    library = ":cli",
    args = [
        "--shipshape_test_docker_tag=staging",
    ],
)

# NOTE: Make sure to rebuild your docker images before running these tests!
#   $ bazel build //shipshape/docker:service //shipshape/androidlint_analyzer/docker:android_lint
go_test(
    name = "test_local",
    srcs = [
        "shipshape_test.go",
    ],
    # Because the docker images are manual, we want to make this
    # manual as well or we'll end up building the docker images
    # from a wildcard build.
    tags = ["manual"],
    data = [
        "testdata",
        # TODO(rsk): figure out why we need to rely on the output
        # marker rather than the target name (may be a bug in
        # either Bazel or the docker_build extension).
        "//shipshape/docker:service.done",
        "//shipshape/androidlint_analyzer/docker:android_lint.done",
    ],
    library = ":cli",
    deps = [
        "//shipshape/util/docker:docker",
    ],
    args = [
        "--shipshape_test_docker_tag=local",
    ],
)

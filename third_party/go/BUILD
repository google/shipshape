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

load("/tools/build_rules/go", "go_binary", "go_library")
load("build", "package")

# TODO split libraries into separate BUILD files once https://github.com/google/bazel/issues/118 is fixed
licenses(["notice"])

go_binary(
    name = "protoc-gen-go",
    srcs = [
        "src/github.com/golang/protobuf/protoc-gen-go/link_grpc.go",
        "src/github.com/golang/protobuf/protoc-gen-go/main.go",
    ],
    deps = [
        ":protobuf",
        ":protoc-gen-go_generator",
        ":protoc-gen-go_internal_grpc",
    ],
)

package(
    name = "protoc-gen-go_generator",
    package = "github.com/golang/protobuf/protoc-gen-go/generator",
    deps = [
        ":protobuf",
        ":protoc-gen-go_descriptor",
        ":protoc-gen-go_plugin",
    ],
)

package(
    name = "protoc-gen-go_plugin",
    package = "github.com/golang/protobuf/protoc-gen-go/plugin",
    deps = [
        ":protobuf",
        ":protoc-gen-go_descriptor",
    ],
)

package(
    name = "protoc-gen-go_descriptor",
    package = "github.com/golang/protobuf/protoc-gen-go/descriptor",
    deps = [
        ":protobuf",
    ],
)

package(
    name = "protoc-gen-go_internal_grpc",
    package = "github.com/golang/protobuf/protoc-gen-go/internal/grpc",
    deps = [
        ":protoc-gen-go_descriptor",
        ":protoc-gen-go_generator",
    ],
)

package(
    name = "protobuf",
    exclude_srcs = ["*_unsafe.go"],
    package = "github.com/golang/protobuf/proto",
)

package(
    name = "go-yaml",
    package = "gopkg.in/yaml.v2",
)

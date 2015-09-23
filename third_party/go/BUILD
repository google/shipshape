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

# TODO split libraries into separate BUILD files once https://github.com/google/bazel/issues/118 is fixed
licenses(["notice"])

go_binary(
    name = "protoc-gen-go",
    srcs = [
        "github.com/src/github.com/golang/protobuf/protoc-gen-go/link_grpc.go",
        "github.com/src/github.com/golang/protobuf/protoc-gen-go/main.go",
    ],
    deps = [
        ":protobuf",
        ":protoc-gen-go_generator",
        ":protoc-gen-go_internal_grpc",
    ],
)

go_library(
    name = "protoc-gen-go_generator",
    srcs = [
        "github.com/src/github.com/golang/protobuf/protoc-gen-go/generator/generator.go",
    ],
    package = "github.com/golang/protobuf/protoc-gen-go/generator",
    deps = [
        ":protobuf",
        ":protoc-gen-go_descriptor",
        ":protoc-gen-go_plugin",
    ],
)

go_library(
    name = "protoc-gen-go_plugin",
    srcs = [
        "github.com/src/github.com/golang/protobuf/protoc-gen-go/plugin/plugin.pb.go",
    ],
    package = "github.com/golang/protobuf/protoc-gen-go/plugin",
    deps = [
        ":protobuf",
        ":protoc-gen-go_descriptor",
    ],
)

go_library(
    name = "protoc-gen-go_descriptor",
    srcs = [
        "github.com/src/github.com/golang/protobuf/protoc-gen-go/descriptor/descriptor.pb.go",
    ],
    package = "github.com/golang/protobuf/protoc-gen-go/descriptor",
    deps = [
        ":protobuf",
    ],
)

go_library(
    name = "protoc-gen-go_internal_grpc",
    srcs = [
        "github.com/src/github.com/golang/protobuf/protoc-gen-go/internal/grpc/grpc.go",
    ],
    package = "github.com/golang/protobuf/protoc-gen-go/internal/grpc",
    deps = [
        ":protoc-gen-go_descriptor",
        ":protoc-gen-go_generator",
    ],
)

go_library(
    name = "protobuf",
    srcs = [
        "github.com/src/github.com/golang/protobuf/proto/clone.go",
        "github.com/src/github.com/golang/protobuf/proto/decode.go",
        "github.com/src/github.com/golang/protobuf/proto/encode.go",
        "github.com/src/github.com/golang/protobuf/proto/equal.go",
        "github.com/src/github.com/golang/protobuf/proto/extensions.go",
        "github.com/src/github.com/golang/protobuf/proto/lib.go",
        "github.com/src/github.com/golang/protobuf/proto/message_set.go",
        "github.com/src/github.com/golang/protobuf/proto/pointer_reflect.go",
        "github.com/src/github.com/golang/protobuf/proto/properties.go",
        "github.com/src/github.com/golang/protobuf/proto/text.go",
        "github.com/src/github.com/golang/protobuf/proto/text_parser.go",
    ],
    package = "github.com/golang/protobuf/proto",
)

go_library(
    name = "go-yaml",
    srcs = [
        "gopkg.in/src/gopkg.in/yaml.v2/apic.go",
        "gopkg.in/src/gopkg.in/yaml.v2/decode.go",
        "gopkg.in/src/gopkg.in/yaml.v2/emitterc.go",
        "gopkg.in/src/gopkg.in/yaml.v2/encode.go",
        "gopkg.in/src/gopkg.in/yaml.v2/parserc.go",
        "gopkg.in/src/gopkg.in/yaml.v2/readerc.go",
        "gopkg.in/src/gopkg.in/yaml.v2/resolve.go",
        "gopkg.in/src/gopkg.in/yaml.v2/scannerc.go",
        "gopkg.in/src/gopkg.in/yaml.v2/sorter.go",
        "gopkg.in/src/gopkg.in/yaml.v2/writerc.go",
        "gopkg.in/src/gopkg.in/yaml.v2/yaml.go",
        "gopkg.in/src/gopkg.in/yaml.v2/yamlh.go",
        "gopkg.in/src/gopkg.in/yaml.v2/yamlprivateh.go",
    ],
    package = "gopkg.in/yaml.v2",
)

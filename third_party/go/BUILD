load("//tools/build_rules/go", "go_binary", "go_library")

package(default_visibility = ["//shipshape:default_visibility"])

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

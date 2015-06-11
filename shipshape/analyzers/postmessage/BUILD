package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library")

go_library(
    name = "postmessage",
    srcs = [
        "post_message_analyzer.go",
    ],
    deps = [
        "//shipshape/api:api",
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/util/rpc/client:client",
        "//third_party/go:protobuf",
    ],
)

package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library")

go_library(
    name = "test",
    srcs = [
        "test_util.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/util/rpc/client:client",
        "//shipshape/util/rpc/server:server",
        "//third_party/go:protobuf",
    ],
)

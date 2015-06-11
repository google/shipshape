package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_binary")

go_binary(
    name = "shipshape",
    srcs = [
        "shipshape.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
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

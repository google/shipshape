package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/genproto", "proto_library")
load("//tools/build_rules/go", "go_library", "go_test")

proto_library(
    name = "test_proto",
    src = "test.proto",
    gen_go = 1,
)

go_library(
    name = "server",
    srcs = [
        "endpoint.go",
        "service.go",
    ],
    deps = [
        ":test_proto_go",
        "//shipshape/util/httpencoding:httpencoding",
        "//shipshape/util/rpc/protocol:protocol",
    ],
)

go_test(
    name = "server_test",
    srcs = [
        "krpc_test.go",
    ],
    deps = [
        ":test_proto_go",
    ],
    library = ":server",
)

package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library", "go_test")

go_library(
    name = "api",
    srcs = [
        "analyzer.go",
        "dispatcher.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/util/file:file",
        "//shipshape/util/rpc/server:server",
        "//shipshape/util/strings:strings",
        "//third_party/go:protobuf",
    ],
)
go_test(
    name = "api_test",
    srcs = [
        "dispatcher_test.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/util/strings:strings",
    ],
    library = ":api",
)

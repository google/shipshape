package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library", "go_binary", "go_test")

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
    library = ":androidlint",
)

package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library", "go_binary")

go_library(
    name = "client",
    srcs = [
        "client.go",
    ],
    deps = [
        "//shipshape/util/httpencoding:httpencoding",
        "//shipshape/util/rpc/protocol:protocol",
    ],
)

go_binary(
    name = "example",
    srcs = [
        "example.go",
    ],
    deps = [
        ":client",
    ],
)

load("//tools/build_rules/go", "go_library", "go_binary")

package(default_visibility = ["//shipshape:default_visibility"])

go_library(
    name = "client",
    srcs = ["client.go"],
    deps = [
        "//shipshape/util/httpencoding",
        "//shipshape/util/rpc/protocol",
    ],
)

go_binary(
    name = "example",
    srcs = ["example.go"],
    deps = [":client"],
)

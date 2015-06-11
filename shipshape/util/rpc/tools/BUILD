package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_binary")

go_binary(
    name = "unwrap_results",
    srcs = [
        "unwrap_results.go",
    ],
    deps = [
        "//shipshape/util/rpc/client:client",
    ],
)

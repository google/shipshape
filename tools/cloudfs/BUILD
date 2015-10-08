package(default_visibility = ["//shipshape:default_visibility"])

load("/tools/build_rules/go", "go_binary")

go_binary(
    name = "copy_cloud_repo",
    srcs = [
        "copy_cloud_repo.go",
    ],
    deps = [
        "//shipshape/util/file:file",
        "//shipshape/util/rpc/client:client",
        "//third_party/go:protobuf",
    ],
)

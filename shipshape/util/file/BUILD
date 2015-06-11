package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library", "go_binary")

go_library(
    name = "file",
    srcs = [
        "cloud_repo_file_utility.go",
        "file_util.go",
    ],
    deps = [
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/proto:source_context_proto_go",
        "//third_party/go:protobuf",
    ],
)

go_binary(
    name = "copy_cloud_repo",
    srcs = [
        "copy_cloud_repo.go",
    ],
    deps = [
        ":file",
        "//shipshape/util/rpc/client:client",
        "//third_party/go:protobuf",
    ],
)

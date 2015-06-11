package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("//tools/build_rules/go", "go_library")

go_library(
    name = "kindex",
    srcs = [
        "kindex.go",
    ],
    deps = [
        "//third_party/go:protobuf",
        "//third_party/kythe/go/platform/analysis:analysis",
        "//third_party/kythe/go/platform/delimited:delimited",
        "//third_party/kythe/proto:analysis_proto_go",
    ],
)

package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library", "go_test")

go_library(
    name = "govet",
    srcs = [
        "govet_analyzer.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:textrange_proto_go",
        "//third_party/go:protobuf",
    ],
)

go_test(
    name = "govet_test",
    srcs = [
        "govet_analyzer_test.go",
    ],
    deps = [
        "//shipshape/proto:shipshape_context_proto_go",
    ],
    library = ":govet",
)

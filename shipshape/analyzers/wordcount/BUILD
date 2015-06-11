package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library", "go_test")

go_library(
    name = "wordcount",
    srcs = [
        "word_count_analyzer.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//third_party/go:protobuf",
    ],
)

go_test(
    name = "wordcount_test",
    srcs = [
        "word_count_analyzer_test.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/util/test:test",
        "//third_party/go:protobuf",
    ],
    library = ":wordcount",
)

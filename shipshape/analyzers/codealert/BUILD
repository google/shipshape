package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library", "go_test")

go_library(
    name = "codealert",
    srcs = [
        "code_alert_analyzer.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//third_party/go:protobuf",
    ],
)

go_test(
    name = "codealert_test",
    srcs = [
        "code_alert_analyzer_test.go",
    ],
    library = ":codealert",
)

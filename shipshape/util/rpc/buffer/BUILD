package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_library", "go_test")

go_library(
    name = "buffer",
    srcs = [
        "buffer.go",
    ],
)

go_test(
    name = "buffer_test",
    srcs = [
        "buffer_test.go",
    ],
    library = ":buffer",
)

java_test(
    name = "tests",
    srcs = glob(["*Test.java"]),
    deps = [
        "//shipshape/java/com/google/shipshape/analyzers",
        "//shipshape/proto:note_proto_java",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:source_context_proto_java",
        "//shipshape/proto:textrange_proto_java",
        "//third_party/guava",
        "//third_party/junit",
        "//third_party/hamcrest_all",
    ],
    data = [
        ":testdata",
        "//third_party/checkstyle",
    ],
)

filegroup(
    name = "testdata",
    srcs = glob(["testdata/**/*.*"]),
)

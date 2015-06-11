package(default_visibility = ["//shipshape:default_visibility"])

java_library(
    name = "analyzers",
    srcs = [
        "EncodingOffsetConverter.java",
        "ErrorProneAnalyzer.java",
        "FixAndReplacementConverter.java",
        "JavacAnalyzer.java",
        "PostMessageAnalyzer.java",
    ],
    deps = [
        "//shipshape/java/com/google/shipshape/service:api",
        "//shipshape/proto:note_proto_java",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:source_context_proto_java",
        "//shipshape/proto:textrange_proto_java",
        "//third_party/error-prone:error-prone",
        "//third_party/guava:guava",
        "//third_party/jsr305_annotations:jsr305",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/java:java",
        "//third_party/kythe/proto:analysis_proto_java",
        "//third_party/kythe/proto:storage_proto_java",
        "//third_party/openjdk:tools_jar",
        "//third_party/proto:protobuf_java",
    ],
)

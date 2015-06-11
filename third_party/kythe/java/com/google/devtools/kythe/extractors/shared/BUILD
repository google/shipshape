package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

java_library(
    name = "shared",
    srcs = [
        "CompilationDescription.java",
        "CompilationFileInputComparator.java",
        "ExtractionException.java",
        "ExtractorUtils.java",
        "IndexInfoUtils.java",
    ],
    deps = [
        "//third_party/guava:guava",
        "//third_party/kythe/java/com/google/devtools/kythe/common:common",
        "//third_party/kythe/proto:analysis_proto_java",
        "//third_party/kythe/proto:storage_proto_java",
        "//third_party/proto:protobuf_java",
    ],
)

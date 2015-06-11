package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

java_library(
    name = "java",
    srcs = [
        "JavaCompilationDetails.java",
        "JavacAnalysisDriver.java",
        "JavacAnalyzer.java",
        "JavacOptionsUtils.java",
    ],
    deps = [
        "//third_party/guava:guava",
        "//third_party/jsr305_annotations:jsr305",
        "//third_party/kythe/java/com/google/devtools/kythe/common:common",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/java/filemanager:filemanager",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/shared:shared",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/shared:file_data",
        "//third_party/kythe/proto:analysis_proto_java",
        "//third_party/kythe/proto:storage_proto_java",
        "//third_party/openjdk:tools_jar",
        "//third_party/proto:protobuf_java",
    ],
)

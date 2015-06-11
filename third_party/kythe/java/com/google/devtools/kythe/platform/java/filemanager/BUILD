package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

java_library(
    name = "filemanager",
    srcs = [
        "CompilationUnitBasedJavaFileManager.java",
        "CompilationUnitBasedJavaFileStore.java",
        "CompilationUnitFileTree.java",
        "CustomFileObject.java",
        "CustomJavaFileObject.java",
        "JavaFileStore.java",
        "JavaFileStoreBasedFileManager.java",
    ],
    deps = [
        "//third_party/guava:guava",
        "//third_party/kythe/java/com/google/devtools/kythe/common:common",
        "//third_party/kythe/java/com/google/devtools/kythe/platform/shared:file_data",
        "//third_party/kythe/proto:analysis_proto_java",
        "//third_party/proto:protobuf_java",
    ],
)

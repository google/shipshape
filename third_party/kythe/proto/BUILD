load("rules/proto", "proto_library")

package(default_visibility = ["//kythe:internal"])

proto_library(
    name = "analysis",
    srcs = ["analysis.proto"],
    go_api_version = 2,
    java_api_version = 2,
    deps = [
        ":storage",
    ],
)

proto_library(
    name = "storage",
    srcs = ["storage.proto"],
    go_api_version = 2,
    java_api_version = 2,
)

proto_library(
    name = "xref",
    srcs = ["xref.proto"],
    has_services = 1,
    go_api_version = 2,
    java_api_version = 2,
)

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("//tools/build_rules/genproto", "proto_library")

proto_library(
    name = "any_proto",
    src = "any.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "storage_proto",
    src = "storage.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "analysis_proto",
    src = "analysis.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":any_proto",
        ":storage_proto",
    ],
)

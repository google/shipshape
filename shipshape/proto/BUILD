package(default_visibility = ["//shipshape:default_visibility"])

load("/tools/build_rules/genproto", "proto_library")

proto_library(
    name = "source_context_proto",
    src = "source_context.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "textrange_proto",
    src = "textrange.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "note_proto",
    src = "note.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":source_context_proto",
        ":textrange_proto",
    ],
)

proto_library(
    name = "shipshape_context_proto",
    src = "shipshape_context.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":source_context_proto",
        "//third_party/kythe/proto:analysis_proto",
    ],
)

proto_library(
    name = "shipshape_rpc_proto",
    src = "shipshape_rpc.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":note_proto",
        ":shipshape_context_proto",
    ],
)

proto_library(
    name = "shipshape_config_proto",
    src = "shipshape_config.proto",
    gen_go = 1,
    gen_java = 1,
)

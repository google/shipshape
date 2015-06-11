package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_binary", "go_library", "go_test")

go_binary(
    name = "go_dispatcher",
    srcs = [
        "go_dispatcher.go",
    ],
    deps = [
        "//shipshape/analyzers/codealert:codealert",
        "//shipshape/analyzers/govet:govet",
        "//shipshape/analyzers/jshint:jshint",
        "//shipshape/analyzers/postmessage:postmessage",
        "//shipshape/analyzers/pylint:pylint",
        "//shipshape/analyzers/wordcount:wordcount",
        "//shipshape/api:api",
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/util/rpc/server:server",
        "//third_party/go:protobuf",
    ],
)

go_library(
    name = "service",
    srcs = [
        "config.go",
        "driver.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_config_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/util/file:file",
        "//shipshape/util/rpc/client:client",
        "//shipshape/util/rpc/server:server",
        "//shipshape/util/strings:strings",
        "//third_party/go:protobuf",
        "//third_party/go:go-yaml",
        "//third_party/kythe/go/platform/kindex:kindex",
        "//third_party/kythe/proto:analysis_proto_go",
    ],
)

go_binary(
    name = "shipshape",
    srcs = [
        "shipshape.go",
    ],
    deps = [
        ":service",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/util/rpc/server:server",
        "//third_party/go:protobuf",
    ],
)

go_test(
    name = "service_test",
    srcs = [
        "config_test.go",
        "driver_test.go",
    ],
    deps = [
        "//shipshape/proto:note_proto_go",
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/util/rpc/server:server",
        "//shipshape/util/test:test",
        "//third_party/go:protobuf",
    ],
    library = ":service",
)

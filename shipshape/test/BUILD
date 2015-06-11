package(default_visibility = ["//shipshape:default_visibility"])

load("//tools/build_rules/go", "go_binary")

go_binary(
    name = "test_shipshape_client",
    srcs = [
        "test_shipshape_client.go",
    ],
    deps = [
        "//shipshape/proto:shipshape_context_proto_go",
        "//shipshape/proto:shipshape_rpc_proto_go",
        "//shipshape/proto:source_context_proto_go",
        "//shipshape/util/file:file",
        "//shipshape/util/rpc/client:client",
        "//third_party/go:protobuf",
    ],
)

java_binary(
    name = "test_request_via_stream",
    srcs = [
        "TestRequestViaStream.java",
    ],
    deps = [
        "//shipshape/proto:note_proto_java",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:shipshape_rpc_proto_java",
        "//shipshape/proto:source_context_proto_java",
        "//third_party/proto:protobuf_java",
    ],
    main_class = "test.TestRequestViaStream",
)

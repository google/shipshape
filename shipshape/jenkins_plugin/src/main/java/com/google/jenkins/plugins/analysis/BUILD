package(default_visibility = ["//shipshape:default_visibility"])

java_library(
    name = "shipshape",
    srcs = [
        "AnalysisRunner.java",
        "ShipshapeException.java",
        "ShipshapeSlave.java",
        "ShipshapeStreamLogger.java",
        "StreamBufferHandler.java",
    ],
    deps = [
        "//shipshape/java/com/google/shipshape/util/rpc:client",
        "//shipshape/proto:note_proto_java",
        "//shipshape/proto:shipshape_context_proto_java",
        "//shipshape/proto:shipshape_rpc_proto_java",
        "//shipshape/proto:source_context_proto_java",
        "//shipshape/proto:textrange_proto_java",
        "//third_party/guava:guava",
        "//third_party/jenkins:access-modifier-annotation",
        "//third_party/jenkins:bridge-method-annotation",
        "//third_party/jenkins:bytecode-compatibility-transformer",
        "//third_party/jenkins:jenkins-core",
        "//third_party/jenkins:json-lib",
        "//third_party/jenkins:remoting",
        "//third_party/jenkins:servlet-api",
        "//third_party/jenkins:stapler",
        "//third_party/proto:protobuf_java",
    ],
)

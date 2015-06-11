package(default_visibility = ["//shipshape:default_visibility"])

java_library(
    name = "protocol",
    javacopts = [
        "-source 8",
        "-target 8",
    ],
    srcs = [
        "Context.java",
        "Method.java",
        "OutputChannel.java",
        "Protocol.java",
        "Service.java",
    ],
    deps = [
        "//third_party/gson:gson",
        "//third_party/gson:proto",
        "//third_party/guava:guava",
        "//third_party/proto:protobuf_java",
    ],
)
java_library(
    name = "server",
    javacopts = [
        "-source 8",
        "-target 8",
    ],
    srcs = [
        "HttpServerFrontend.java",
        "Server.java",
        "ServerException.java",
    ],
    deps = [
        ":protocol",
        "//third_party/gson:gson",
        "//third_party/guava:guava",
    ],
)
java_library(
    name = "client",
    javacopts = [
        "-source 8",
        "-target 8",
    ],
    srcs = [
        "Connection.java",
        "HttpTransport.java",
    ],
    deps = [
        ":protocol",
        ":server",
        "//third_party/gson:gson",
        "//third_party/guava:guava",
    ],
)

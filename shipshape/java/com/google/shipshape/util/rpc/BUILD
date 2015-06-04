package(default_visibility = ["//shipshape:default_visibility"])

java_library(
    name = "server",
    srcs = [
        "HttpServerFrontend.java",
        "Server.java",
        "ServerException.java",
    ],
    javacopts = ["-source 1.8 -target 1.8"],
    deps = [
        ":protocol",
        "//third_party/gson",
        "//third_party/guava",
    ],
)

java_library(
    name = "client",
    srcs = [
        "Connection.java",
        "HttpTransport.java",
    ],
    javacopts = ["-source 1.8 -target 1.8"],
    deps = [
        ":protocol",
        ":server",
        "//third_party/gson",
        "//third_party/guava",
    ],
)

java_library(
    name = "protocol",
    srcs = [
        "Context.java",
        "Method.java",
        "OutputChannel.java",
        "Protocol.java",
        "Service.java",
    ],
    javacopts = ["-source 1.8 -target 1.8"],
    deps = [
        "//third_party/gson",
        "//third_party/gson:proto",
        "//third_party/guava",
        "//third_party/proto:protobuf_java",
    ],
)

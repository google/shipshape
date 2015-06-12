load("//tools/build_rules/docker", "docker_build")

docker_build(
    name = "service",
    src = "Dockerfile",
    image_name = "gcr.io/shipshape_releases/service",
    tags = ["manual"],
    use_cache = True,
    data = [
        "//shipshape/java/com/google/shipshape/service:java_dispatcher_deploy.jar",
        "//shipshape/java/com/google/shipshape/service:javac_dispatcher_deploy.jar",
        "//shipshape/service:go_dispatcher",
        "//shipshape/service:shipshape",
        "endpoint.sh",
    ],
)


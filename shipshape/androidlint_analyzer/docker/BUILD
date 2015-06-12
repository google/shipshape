load("//tools/build_rules/docker", "docker_build")

docker_build(
    name = "android_lint",
    src = "Dockerfile",
    image_name = "gcr.io/shipshape_releases/android_lint",
    tags = ["manual"],
    use_cache = True,
    data = [
        "endpoint.sh",
        "//shipshape/androidlint_analyzer/androidlint:android_lint_service",
    ],
)

load("/tools/build_rules/go", "go_package")

# Simple wrapper around go_package for third_party/go libraries.
def package(name, package, visibility=None,
            deps=[], exclude_srcs=[]):
  go_package(
    name = name,
    package = package,
    srcs = "src/"+package,
    exclude_srcs = exclude_srcs,
    deps = deps,
    tests = 0,
    visibility = visibility,
  )

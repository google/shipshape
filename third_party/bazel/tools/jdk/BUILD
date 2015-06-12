# Copyright 2015 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

filegroup(
    name = "java",
    srcs = ["//external:java"],
)

filegroup(
    name = "bootclasspath",
    srcs = ["//external:bootclasspath"],
)

filegroup(
    name = "extdir",
    srcs = ["//external:extdir"],
)

filegroup(
    name = "langtools",
    srcs = ["//external:langtools"],
)

java_import(
    name = "langtools-neverlink",
    jars = ["//external:langtools"],
    neverlink = 1,
)

# This one is just needed because of how filegroup redirection works.
filegroup(name = "jdk-null")

filegroup(
    name = "jdk",
    srcs = [
        ":jdk-null",
        "//external:jdk-default",
    ],
)

java_toolchain(
    name = "toolchain",
    encoding = "UTF-8",
    source_version = "7",
    target_version = "7",
)

exports_files([
    "JavaBuilder_deploy.jar",
    "SingleJar_deploy.jar",
])

filegroup(
    name = "ijar",
    srcs = ["//third_party/ijar"],
)

filegroup(
    name = "srcs",
    srcs = ["BUILD"],  # Tools are build from the workspace for tests.
)

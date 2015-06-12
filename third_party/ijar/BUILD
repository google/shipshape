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

cc_library(
    name = "zip",
    srcs = ["zip.cc"],
    hdrs = [
        "common.h",
        "zip.h",
    ],
    copts = ["-Wno-unused-private-field"],
    deps = [
        "//third_party/zlib",
    ],
)

cc_binary(
    name = "zipper",
    srcs = ["zip_main.cc"],
    deps = [":zip"],
)

cc_binary(
    name = "ijar",
    srcs = [
        "classfile.cc",
        "ijar.cc",
    ],
    copts = ["-Wno-unused-private-field"],
    deps = [":zip"],
)

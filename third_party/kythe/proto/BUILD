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

load("//tools/build_rules/genproto", "proto_library")

proto_library(
    name = "any_proto",
    src = "any.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "storage_proto",
    src = "storage.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "analysis_proto",
    src = "analysis.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":any_proto",
        ":storage_proto",
    ],
)

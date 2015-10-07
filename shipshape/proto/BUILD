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

package(default_visibility = ["//shipshape:default_visibility"])

load("/tools/build_rules/genproto", "proto_library")

proto_library(
    name = "source_context_proto",
    src = "source_context.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "textrange_proto",
    src = "textrange.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "note_proto",
    src = "note.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":source_context_proto",
        ":textrange_proto",
    ],
)

proto_library(
    name = "shipshape_context_proto",
    src = "shipshape_context.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":source_context_proto",
#        "//third_party/kythe/proto:analysis_proto",
    ],
)

proto_library(
    name = "shipshape_rpc_proto",
    src = "shipshape_rpc.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":note_proto",
        ":shipshape_context_proto",
    ],
)

proto_library(
    name = "shipshape_config_proto",
    src = "shipshape_config.proto",
    gen_go = 1,
    gen_java = 1,
)

proto_library(
    name = "shipshape_reporter_proto",
    src = "shipshape_reporter.proto",
    gen_go = 1,
    gen_java = 1,
    deps = [
        ":note_proto",
        ":shipshape_context_proto",
    ],
)

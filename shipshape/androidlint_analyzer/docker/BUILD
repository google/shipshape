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

load("/tools/build_rules/docker", "docker_build")

package(default_visibility = ["//shipshape:default_visibility"])

docker_build(
    name = "android_lint",
    src = "Dockerfile",
    image_name = "beta.gcr.io/shipshape_releases/android_lint:local",
    tags = ["manual"],
    use_cache = True,
    data = [
        "endpoint.sh",
        "//shipshape/androidlint_analyzer/androidlint:android_lint_service",
    ],
)

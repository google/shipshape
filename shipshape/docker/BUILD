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


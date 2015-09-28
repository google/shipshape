#!/bin/bash

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

# Script that build a test Docker image for Shipshape and then starts a
# container using the image.

set -eu

declare -xr TEST_DIR=$(realpath $(dirname "$0"))
declare -xr CONTAINER="test_testing_env"
declare -xr IMAGE="gcr.io/shipshape_releases/testing_env"

"${TEST_DIR}/docker_build.sh" "${CONTAINER}" "${IMAGE}" || exit 1

# Starts a container with the test image and a volume mapped to /tmp.
echo " Starting docker container ... "
docker run --privileged -it --name "${CONTAINER}" -v /tmp:/tmp "${IMAGE}" bash


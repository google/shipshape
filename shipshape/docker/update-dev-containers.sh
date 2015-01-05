#!/bin/bash

# Copyright 2014 Google Inc. All rights reserved.
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

set -eux

CONVOY_BUCKET=dev_containers
CONVOY_SERVER=gcr.io
DEFAULT_TAG=latest

DEV_IMAGES=(
  google/cloud-dev-java:prod
  google/cloud-dev-nodejs:prod
)

gcloud preview docker --server=$CONVOY_SERVER --authorize_only

ensure_tag() {
  local image="$1"
  if [[ "$image" != *:* ]]; then
    image="${image}:${DEFAULT_TAG}"
  fi
  echo "$image"
}

remove_repo_name() {
  sed -r 's#(.*/)?(.*)#\2#' <<<"$1"
}

convoy_url() {
  echo "${CONVOY_SERVER}/_b_${CONVOY_BUCKET}/$(remove_repo_name "$1")"
}

function pull_image() {
  local image="$(ensure_tag "$1")"
  echo Pulling $image
  local repo="$(convoy_url "$image")"

  if ! gcloud preview docker pull "$repo"; then
    echo "Failed pulling: $repo" >&2
    exit 1
  fi

  if ! gcloud preview docker tag "$repo" "$image"; then
    echo "Failed tagging: $image" >&2
    exit 1
  fi
}

for img in ${DEV_IMAGES[@]}; do
  pull_image "$img"
done


#!/bin/bash -e

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

# This script is only called from server.js, see that file for usage.
# directory to clone/sync repo/patch in (/repo/repo_<id>)
REPO_DIR=$1
# the diff id from phabricator, a unique id in phabricator.
DIFF_ID=$2
# the file that server.js expects the json output from shipshape to be
# placed.
JSON_OUTPUT=$3
# If the repo dir exists, checkout master & pull head
# else, clone the repo
if [ -d "$REPO_DIR" ]; then
  cd $REPO_DIR
  git checkout master
  git pull
else
  git clone $GIT_REPO $REPO_DIR
  cd $REPO_DIR
fi
# arc patch will apply the revision in a new branch
arc patch --update --force --diff $DIFF_ID
/shipshape/cli/shipshape --categories="$SHIPSHAPE_CATEGORIES" \
  --json_output=$JSON_OUTPUT $REPO_DIR
#TODO(jvg): delete patch branch

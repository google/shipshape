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

# Script that sets up a test repo and then runs the Shipshape CLI
# on that test repo.

set -eu

TEST_DIR=$(realpath $(dirname "$0"))
BASE_DIR=$(realpath "${TEST_DIR}/../..")
CAMPFIRE="${BASE_DIR}/campfire"
CAMPFIRE_OUT="${BASE_DIR}/campfire-out"

CONVOY_URL='gcr.io'
LOCAL_WORKSPACE='/tmp/shipshape-tests'
LOG_FILE='end_to_end_test.log'
REPO=$CONVOY_URL/shipshape_releases
KYTHE_TEST='false'
CONTAINERS=(
  //shipshape/docker:service
  //shipshape/androidlint_analyzer/docker:android_lint
)

# Check tag argument.
[[ "$#" == 1 ]] || [[ "$#" == 2 ]] || { echo "Usage: ./end-to-end-test.sh <TAG> [IS_KYTHE_TEST]" 1>&2 ; exit 1; }

TAG=${1,,} # make lower case
IS_LOCAL_RUN=false; [[ "$TAG" == "local" ]] && IS_LOCAL_RUN=true
echo $IS_LOCAL_RUN

[[ "$#" == 2 ]] && KYTHE_TEST=${2,,}
echo $KYTHE_TEST

# Build repo in local mode
if [[ "$IS_LOCAL_RUN" == true ]]; then
  echo "Running with locally built containers"
  $CAMPFIRE clean
  $CAMPFIRE build //shipshape/cli/...
  for container in ${CONTAINERS[@]}; do
    echo 'Building and deploying '$container' ...'
    $CAMPFIRE package --start_registry=false --docker_tag=$TAG $container
    IFS=':' # Set global string separator so we can split the image name
    names=(${container[@]})
    name=${names[1]}
    docker tag -f $name:$TAG $REPO/$name:$TAG
    IFS=' ' # reset it back to a space
  done
fi

# Set up log file
touch $LOG_FILE
rm $LOG_FILE
echo ">>> Detailed output will appear in $LOG_FILE"

# Create a test repository to run analysis on
rm -r $LOCAL_WORKSPACE || true
mkdir -p $LOCAL_WORKSPACE
echo "this is not javascript" > $LOCAL_WORKSPACE/test.js
mkdir -p $LOCAL_WORKSPACE/src/main/java/com/google/shipshape/
cat <<'EOF' >> $LOCAL_WORKSPACE/src/main/java/com/google/shipshape/App.java
package com.google.shipshape;

/**
 * Hello world!
 *
 */
public class App
{
  private String str;
  public App(String str) {
    str = str;
  }
  public static void main( String[] args )
  {
    if (args[0] == "Hello");
    System.out.printf("Hello World!");
  }
}
EOF
cat <<'EOF' >> $LOCAL_WORKSPACE/pom.xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>testtesttest</groupId>
  <artifactId>test-app</artifactId>
  <version>1.0-SNAPSHOT</version>
</project>
EOF

# Run CLI over the new repo
echo "---- Running CLI over test repo" &>> $LOG_FILE
$CAMPFIRE_OUT/bin/shipshape/cli/shipshape --tag=$TAG --categories='PostMessage,JSHint,ErrorProne' --build=maven --stderrthreshold=INFO --local_kythe=$KYTHE_TEST $LOCAL_WORKSPACE >> $LOG_FILE
echo "Analysis complete, checking results..."
# Run a second time for AndroidLint. We have to do this separately because
# otherwise kythe will try to build all the java files, even the ones that maven
# doesn't build.
cp -r $BASE_DIR/shipshape/androidlint_analyzer/test_data/TicTacToeLib $LOCAL_WORKSPACE/
echo "---- Running CLI over test repo, android test" &>> $LOG_FILE
$CAMPFIRE_OUT/bin/shipshape/cli/shipshape --tag=$TAG --analyzer_images=$REPO/android_lint:$TAG --categories='AndroidLint' --stderrthreshold=INFO --local_kythe=$KYTHE_TEST $LOCAL_WORKSPACE >> $LOG_FILE
echo "Analysis complete, checking results..."

# Quick sanity checks of output.
JSHINT_COUNT=$(grep JSHint $LOG_FILE | wc -l)
POSTMESSAGE_COUNT=$(grep PostMessage $LOG_FILE | wc -l)
ERRORPRONE_COUNT=$(grep ErrorProne $LOG_FILE | wc -l)
ANDROIDLINT_COUNT=$(grep AndroidLint $LOG_FILE | wc -l)
FAILURE_COUNT=$(grep Failure $LOG_FILE | wc -l)
TEST_STATUS=0
[[ $JSHINT_COUNT == 8 ]] || { echo "Wrong number of JSHint results, expected 8, found $JSHINT_COUNT" 1>&2 ; TEST_STATUS=1; }
[[ $POSTMESSAGE_COUNT == 1 ]] || { echo "Wrong number of PostMessage results, expected 1, found $POSTMESSAGE_COUNT" 1>&2 ; TEST_STATUS=1; }
[[ $ERRORPRONE_COUNT == 2 ]] || { echo "Wrong number of ErrorProne results, expected 2, found $ERRORPRONE_COUNT" 1>&2 ; TEST_STATUS=1; }
[[ $ANDROIDLINT_COUNT == 8 ]] || { echo "Wrong number of AndroidLint results, expected 9, found $ANDROIDLINT_COUNT" 1>&2 ; TEST_STATUS=1; }
[[ $FAILURE_COUNT == 0 ]] || { echo "Some analyses failed; please check $LOG_FILE" 1>&2 ; TEST_STATUS=1; }

if [[ $TEST_STATUS -eq 0 ]]; then
  echo "Success! Analyzer produced expected number of results. Full output in $LOG_FILE"
fi
exit $(($TEST_STATUS))

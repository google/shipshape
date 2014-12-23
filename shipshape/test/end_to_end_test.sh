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

CONVOY_URL='gcr.io'
LOCAL_WORKSPACE='/tmp/shipshape-tests'
LOG_FILE='end_to_end_test.log'
REPO=$CONVOY_URL/_b_shipshape_registry
CONTAINERS=(
  //shipshape/docker/base:base
  //shipshape/docker:service
  //shipshape/androidlint_analyzer/docker:android_lint
)

# Check tag argument.
[[ "$#" == 1 ]] || { echo "Usage: ./end-to-end-test.sh <TAG>" 1>&2 ; exit 1; }

TAG=${1,,} # make lower case
IS_LOCAL_RUN=false; [[ "$TAG" == "local" ]] && IS_LOCAL_RUN=true
echo $IS_LOCAL_RUN

# Build repo in local mode
if [[ "$IS_LOCAL_RUN" == true ]]; then
  echo "Running with locally built containers"
  ../../campfire clean
  ../../campfire build //shipshape/cli/...
  for container in ${CONTAINERS[@]}; do
    echo 'Building and deploying '$container' ...'
    ../../campfire package --start_registry=false --docker_tag=$TAG $container
    IFS=':' # Set global string separator so we can split the image name
    names=(${container[@]})
    name=${names[1]}
    docker tag $name:$TAG $REPO/$name:$TAG
    IFS=' ' # reset it back to a space
  done
fi

# Set up log file
touch $LOG_FILE
rm $LOG_FILE
echo ">>> Detailed output will appear in $LOG_FILE"

# Get access token for the convoy docker registry
gcloud preview docker --server=$CONVOY_URL --authorize_only

# Create a test repository to run analysis on
rm -r $LOCAL_WORKSPACE
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
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/maven-v4_0_0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.google.shipshape</groupId>
  <artifactId>test-app</artifactId>
  <packaging>jar</packaging>
  <version>1.0-SNAPSHOT</version>
  <name>test-app</name>
  <url>http://maven.apache.org</url>
  <dependencies>
    <dependency>
      <groupId>junit</groupId>
      <artifactId>junit</artifactId>
      <version>3.8.1</version>
      <scope>test</scope>
    </dependency>
  </dependencies>
</project>
EOF


# Run CLI over the new repo
echo "---- Running CLI over test repo" &>> $LOG_FILE
../../campfire-out/bin/shipshape/cli/shipshape --tag=$TAG --categories='PostMessage,JSHint,ErrorProne' --build=maven --try_local=$IS_LOCAL_RUN --stderrthreshold=INFO $LOCAL_WORKSPACE >> $LOG_FILE
echo "Analysis complete, checking results..."

# Quick sanity checks of output.
JSHINT_COUNT=$(grep JSHint $LOG_FILE | wc -l)
POSTMESSAGE_COUNT=$(grep PostMessage $LOG_FILE | wc -l)
ERRORPRONE_COUNT=$(grep ErrorProne $LOG_FILE | wc -l)
FAILURE_COUNT=$(grep Failure $LOG_FILE | wc -l)
[[ $JSHINT_COUNT == 8 ]] || { echo "Wrong number of JSHint results, expected 8, found $JSHINT_COUNT" 1>&2 ; exit 1; }
[[ $POSTMESSAGE_COUNT == 1 ]] || { echo "Wrong number of PostMessage results, expected 1, found $POSTMESSAGE_COUNT" 1>&2 ; exit 1; }
[[ $ERRORPRONE_COUNT == 2 ]] || { echo "Wrong number of ErrorProne results, expected 2, found $ERRORPRONE_COUNT" 1>&2 ; exit 1; }
[[ $FAILURE_COUNT == 0 ]] || { echo "Some analyses failed; please check $LOG_FILE" 1>&2 ; exit 1; }
echo "Success! Analyzer produced expected number of results. Full output in $LOG_FILE"

exit 0

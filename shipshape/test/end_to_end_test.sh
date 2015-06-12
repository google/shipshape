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

declare -xr TEST_DIR=$(realpath $(dirname "$0"))
declare -xr BASE_DIR=$(realpath "${TEST_DIR}/../..")
declare -xr SHIPSHAPE="${BASE_DIR}/bazel-bin/shipshape/cli/shipshape"

declare -xr CONVOY_URL='gcr.io'
declare -xr LOCAL_WORKSPACE='/tmp/shipshape-tests'
declare -xr LOG_FILE='end_to_end_test.log'
declare -xr REPO=$CONVOY_URL/shipshape_releases
declare -xr CONTAINERS=(
  //shipshape/docker:service
  //shipshape/androidlint_analyzer/docker:android_lint
)

declare -x KYTHE_TEST=false
declare -x IS_LOCAL_RUN=false
declare -x TAG=''

##############################
# Logs info string
# Globals:
#   None
# Argument:
#   The message to log
# Return:
#   None
##############################
info() {
  echo "INFO: $@" | tee -a $LOG_FILE
}

##############################
# Logs error string
# Globals:
#   None
# Argument:
#   The message to log
# Return:
#   None
##############################
error() {
  echo "ERROR: $@" | tee -a $LOG_FILE
}

##############################
# Runs and logs a command
# Globals:
#   LOG_FILE
# Arguments:
#   Command to run
# Return:
#   None
##############################
run() {
  info "Running command [$@]"
  (`$@`) >> $LOG_FILE 2>&1
}

##############################
# Setup logging
# Globals:
#   LOG_FILE
# Arguments:
#   None
# Return:
#   None
##############################
setup_logging() {
  rm -f $LOG_FILE;
  info "Detailed output will appear in $LOG_FILE"
}

##############################
# Prints help instructions
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
##############################
print_help() {
  echo "USAGE: ./end-to-end-test.sh --tag TAG [--with-kythe]" 1>&2
}

##############################
# Processes script arguments
# Globals:
#   KYTHE_TEST
#   TAG
#   IS_LOCAL_RUN
# Arguments:
#   Script arguments
# Return:
#   None
##############################
process_arguments() {
  while test $# -gt 0; do
    case "$1" in
      -h|--help)
        print_help
        exit 0
        ;;
      --kythe-test)
        info "Including kythe in test"
        KYTHE_TEST=true
        readonly KYTHE_TEST
        shift
        ;;
      --tag)
        shift
        TAG=${1,,} # make lower case
        readonly TAG
        if [[ "$TAG" == "local" ]]; then
          IS_LOCAL_RUN=true
          readonly IS_LOCAL_RUN
        fi
        shift
        ;;
      *)
        error "Unknown argument"
        print_help
        exit 1
        ;;
    esac
  done
  # Make sure we got a tag value
  if [[ -z ${TAG+x} ]]; then
    error "--tag value is missing, TAG=["$TAG"]"
    exit 2
  fi
}

########################################
# Builds and deploys containers locally
# Globals:
#   CAMPFIRE
#   CONTAINERS
#   TAG
#   REPO
# Arguments:
#   None
# Return:
#   None
########################################
build_local() {
  for container in ${CONTAINERS[@]}; do
    info "Building and deploying $container locally ..."
    run bazel build $container
    IFS=':' # Temporarily set global string separator to split image names
    names=(${container[@]})
    name=${names[1]}
    IFS=' ' # reset global string separator
    run "docker tag -f $name:$TAG $REPO/$name:$TAG"
  done
}

#######################################
# Creates a test repository to analyze
# Globals:
#   LOCAL_WORKSPACE
# Arguments:
#   None
# Return:
#   None
#######################################
create_test_repo() {
  info "Creating test repo at $LOCAL_WORKSPACE"
  rm -r "$LOCAL_WORKSPACE" || true
  mkdir -p "$LOCAL_WORKSPACE"
  echo "this is not javascript" > "$LOCAL_WORKSPACE/test.js"
  mkdir -p "$LOCAL_WORKSPACE/src/main/java/com/google/shipshape/"
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
}

#############################################
# Copies Shipshape logs into test log
# Globals:
#   LOG_FILE
# Arguments:
#   Message header for included logs
# Return:
#   None
#############################################
copy_shipshape_logs() {
  log_files=(
    /tmp/shipshape.shipping_container.log
    /tmp/shipshape.go_dispatcher.log
    /tmp/shipshape.java_dispatcher.log
    /tmp/shipshape.javac_dispatcher.log
    /tmp/shipshape.android_lint.log
  )
  info "Copying Shipshape logs into test log ..."
  echo "START[$1]:" >> $LOG_FILE
  for log_file in ${log_files[@]}; do
    echo ""
    echo "LOG_FILE[$log_file]:" >> $LOG_FILE
    if [ -e $log_file ]; then
      cat $log_file >> $LOG_FILE
    else
      echo "log file does not exist" >> $LOG_FILE
    fi
  done
  echo ""
  echo "END[$1]" >> $LOG_FILE
}

#############################################
# Analyzes the test repo
# Globals:
#   LOG_FILE
#   BAZEL_OUT
#   TAG
#   KYTHE_TEST
#   LOCAL_WORKSPACE
# Arguments:
# Return:
#############################################
analyze_test_repo() {
  info "Building Shipshape CLI ..."
  run bazel build shipshape/cli/...
  # Run CLI over the new repo
  info "Analyzing test repo using PostMessage,JSHint,ErrorProne ..."
  "$SHIPSHAPE" --tag=$TAG --categories='PostMessage,JSHint,ErrorProne' --build=maven --stderrthreshold=INFO --local_kythe=$KYTHE_TEST "$LOCAL_WORKSPACE" >> $LOG_FILE 2>&1
  # Copying logs into LOG_FILE to not have them overwritten by the next CLI run
  copy_shipshape_logs 'Logs from first CLI run for PostMessage,JSHint,ErrorProne'
  # Run a second time for AndroidLint. We have to do this separately because
  # otherwise kythe will try to build all the java files, even the ones that maven
  # doesn't build.
  cp -r "$BASE_DIR/shipshape/androidlint_analyzer/test_data/TicTacToeLib" "$LOCAL_WORKSPACE/"
  info "Analyzing test repo using AndroidLint ..."
  "$SHIPSHAPE" --tag=$TAG --analyzer_images=$REPO/android_lint:$TAG --categories='AndroidLint' --stderrthreshold=INFO --local_kythe=$KYTHE_TEST "$LOCAL_WORKSPACE" >> $LOG_FILE 2>&1
  # Copying logs again to LOG_FILE to have all logs in one place
  copy_shipshape_logs 'Logs from second CLI run for AndroidLint'
}

##############################################
# Checks findings
# Globals:
#   LOG_FILE
# Arguments:
#   None
# Return:
#   status of tests
##############################################
check_findings() {
  info "Checking analyzer results ..."
  local jshint=$(grep "\[JSHint\]" $LOG_FILE | wc -l)
  local postmessage=$(grep "\[PostMessage\]" $LOG_FILE | wc -l)
  local errorprone=$(grep "\[ErrorProne\]" $LOG_FILE | wc -l)
  local androidlint=$(grep "\[AndroidLint:" $LOG_FILE | wc -l)
  local failure=$(grep "Failure" $LOG_FILE | wc -l)
  local status=0
  [[ $jshint == 8 ]] || error "Wrong number of JSHint results, expected 8, found $jshint"; status=1;
  [[ $postmessage == 2 ]] || error "Wrong number of PostMessage results, expected 2, found $postmessage"; status=1;
  [[ $errorprone == 2 ]] || error "Wrong number of ErrorProne results, expected 2, found $errorprone"; status=1;
  [[ $androidlint == 8 ]] || error "Wrong number of AndroidLint results, expected 9, found $androidlint"; status=1;
  [[ $failure == 0 ]] || error "Some analyses failed; please check $LOG_FILE" ; status=1;
  if [[ $status -eq 0 ]]; then
    info "Success! Analyzer produced expected number of results. Full output in $LOG_FILE"
  fi
  return $(($status))
}


process_arguments "$@"
setup_logging

# Build repo in local mode
if [[ "$IS_LOCAL_RUN" == true ]]; then
  info "Running with locally built containers"
  build_local
fi

create_test_repo
analyze_test_repo
check_findings


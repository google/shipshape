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

# Script that sets two test repos and then runs the Shipshape CLI
# on those test repo.

set -eu

declare -xr TEST_DIR=$(realpath $(dirname "$0"))
declare -xr BASE_DIR=$(realpath "${TEST_DIR}/../..")
declare -xr SHIPSHAPE="${BASE_DIR}/bazel-bin/shipshape/cli/shipshape"

declare -xr FIRST_REPO='/tmp/shipshape1'
declare -xr SECOND_REPO='/tmp/shipshape2'
declare -xr LOG_FILE='change_dir_test.log'

declare -xr CONVOY_URL='gcr.io'
declare -xr REPO=$CONVOY_URL/shipshape_releases
declare -xr CONTAINERS=(
  //shipshape/docker:service
)

declare -x IS_LOCAL_RUN=false
declare -x TAG=''
declare -x USE_RELEASED_CLI=false

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
  echo "USAGE: ./change_dir_test.sh [--tag TAG] [--released-cli]" 1>&2
}

##############################
# Processes script arguments
# Globals:
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
      --released-cli)
        USE_RELEASED_CLI=true
        shift
        ;;
      *)
        error "Unknown argument"
        print_help
        exit 1
        ;;
    esac
  done
  readonly IS_LOCAL_RUN
  readonly USE_RELEASED_CLI
  # Make sure we got a tag value
  if [[ -z ${TAG+x} ]]; then
    error "--tag value is missing, TAG=["$TAG"]"
    exit 2
  fi
}

########################################
# Builds and deploys containers locally
# Globals:
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
  )
  info "Copying Shipshape logs into test log ..."
  echo "START[$1]:" >> $LOG_FILE
  for log_file in ${log_files[@]}; do
    echo "" >> $LOG_FILE
    echo "LOG_FILE[$log_file]:" >> $LOG_FILE
    if [ -e $log_file ]; then
      cat $log_file >> $LOG_FILE
    else
      echo "log file does not exist" >> $LOG_FILE
    fi
  done
  echo "" >> $LOG_FILE
  echo "END[$1]" >> $LOG_FILE
}

#######################################
# Creates the first test repository
# Globals:
#   FIRST_REPO
# Arguments:
#   None
# Return:
#   None
#######################################
create_first_test_repo() {
  info "Creating test repo at $FIRST_REPO"
  rm -r "$FIRST_REPO" || true
  mkdir -p "$FIRST_REPO"
  echo "var x = 5 var y = 6;" > "$FIRST_REPO/test.js"
}

#######################################
# Creates the second test repository
# Globals:
#   SECOND_REPO
# Arguments:
#   None
# Return:
#   None
#######################################
create_second_test_repo() {
  info "Creating test repo at $SECOND_REPO"
  rm -r "$SECOND_REPO" || true
  mkdir -p "$SECOND_REPO"
  echo "var x = 5; var y = 6" > "$SECOND_REPO/test.js"
}

#############################################
# Analyzes the test repo
# Globals:
#   LOG_FILE
#   SHIPSHAPE
#   TAG
#   USE_RELEASED_CLI
# Arguments:
#   Path to repository to analyze
# Return:
#############################################
analyze_test_repo() {
  if [[ "$USE_RELEASED_CLI" == true ]]; then
    info "Analyzing test repo using JSHint (with the released CLI) ..."
    #gsutil cp gs://shipshape-cli/shipshape /tmp/shipshape-cli
    #chmod a+x /tmp/shipshape-cli
    #/tmp/shipshape-cli --tag=$TAG --categories='JSHint' --stderrthreshold=INFO "$1" >> $LOG_FILE 2>&1
    /google/data/ro/teams/tricorder/shipshape --tag=$TAG --categories='JSHint' --stderrthreshold=INFO "$1" >> $LOG_FILE 2>&1
  else
    run bazel build //...
    info "Analyzing test repo using JSHint (with the locally built CLI) ..."
    "$SHIPSHAPE" --tag=$TAG --categories='JSHint' --stderrthreshold=INFO "$1" >> $LOG_FILE 2>&1
  fi
  # Copying logs into LOG_FILE to not have them overwritten by the next CLI run
  copy_shipshape_logs 'Logs from CLI run for JSHint'
}

##############################################
# Checks first findings
# Globals:
#   LOG_FILE
# Arguments:
#   None
# Return:
#   None
##############################################
check_first_findings() {
  info "Checking first analyzer results ..."
  local jshint_finding=$(grep "Line 1, Col 10 \[JSHint\]" $LOG_FILE | wc -l)
  local failure=$(grep "Failure" $LOG_FILE | wc -l)
  local result=0
  [[ $jshint_finding == 1 ]] || error "Wrong number of JSHint results, expected 1, found $jshint_finding";result=1
  [[ $failure == 0 ]] || error "Some analyses failed; please check $LOG_FILE";result=1
  if [[ $result -eq 0 ]]; then
    info "Success! Analyzer produced expected results. Full output in $LOG_FILE"
  fi
}

##############################################
# Checks second findings
# Globals:
#   LOG_FILE
# Arguments:
#   None
# Return:
#   None
##############################################
check_second_findings() {
  info "Checking second analyzer results ..."
  local jshint_finding=$(grep "Line 1, Col 21 \[JSHint\]" $LOG_FILE | wc -l)
  local failure=$(grep "Failure" $LOG_FILE | wc -l)
  local result=0
  [[ $jshint_finding == 1 ]] || error "Wrong number of JSHint results, expected 1, found $jshint_finding"; result=1;
  [[ $failure == 0 ]] || error "Some analyses failed; please check $LOG_FILE" ; result=1;
  if [[ $result -eq 0 ]]; then
    info "Success! Analyzer produced expected results. Full output in $LOG_FILE"
  fi
}

process_arguments "$@"
setup_logging

# Build repo in local mode
if [[ "$IS_LOCAL_RUN" == true ]]; then
  info "Running with locally built containers"
#  build_local
fi

create_first_test_repo
create_second_test_repo

analyze_test_repo $FIRST_REPO
check_first_findings

analyze_test_repo $SECOND_REPO
check_second_findings


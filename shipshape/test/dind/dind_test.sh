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

# Script that runs the Shipshape CLI in a dind container.

set -eu

declare -xr LOG_FILE='dind_test.log'

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
  echo "USAGE: ./dind_test.sh" 1>&2
}

##############################
# Processes script arguments
# Globals:
#   None
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
      *)
        error "Unknown argument"
        print_help
        exit 1
        ;;
    esac
  done
}

##############################################
# Checks findings
# Globals:
#   LOG_FILE
# Arguments:
#   None
# Return:
#   None
##############################################
check_findings() {
  info "Checking analyzer results ..."
  local postmessage_finding=$(grep "\[PostMessage\]" $LOG_FILE | wc -l)
  local failure=$(grep "Failure" $LOG_FILE | wc -l)
  local result=0
  [[ $postmessage_finding == 2 ]] || error "Wrong number of PostMessage results, expected 2, found $postmessage_finding";result=1
  [[ $failure == 0 ]] || error "Some analyses failed; please check $LOG_FILE";result=1
  if [[ $result -eq 0 ]]; then
    info "Success! Analyzer produced expected results. Full output in $LOG_FILE"
  fi
}

setup_logging

run bazel build shipshape/test/dind/docker:dind-test
docker run --privileged shipshape_dind_test > $LOG_FILE 2>&1

check_findings

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

# Runs the shipshape service along with a set of analyzer services in 
# different TMUX windows for easier local debugging.

set -u # do not set -e; need to catch failures in each window

TMUX_SESSION=shipshape
SHIPSHAPE_BIN=../../campfire-out/bin/shipshape/service/shipshape
GO_DISPATCHER_BIN=../../campfire-out/bin/shipshape/service/go_dispatcher
JAVAC_DISPATCHER_BIN=../../campfire-out/bin/shipshape/java/com/google/shipshape/service/javac_dispatcher

function run_about () {
  cat <<-EOM
  Shipshape is running inside tmux.

  There are multiple processes running each in their own window:
    shipshape: Handles analysis requests, sending to appropriate dispatchers.
    go_dispatcher: Dispatches to all analyses called from go
    javac_dispatcher: Dispatches to javac analyses

  Use <CTRL-B> <p|n> to switch between the different windows.
  Press enter in this window to kill all processes.
EOM
  read
  tmux kill-window -t shipshape
  tmux kill-window -t go_dispatcher
  tmux kill-window -t javac_dispatcher
  tmux kill-window
}

function run_shipshape () {
  echo Running Shipshape server...
  $SHIPSHAPE_BIN --start_service
  if [ $? -ne 0 ] ; then
    echo ERROR: failure running Shipshape binary
    read
  fi
}

function run_go_dispatcher () {
  echo Running go dispatcher server...
  $GO_DISPATCHER_BIN
  if [ $? -ne 0 ] ; then
    echo ERROR: failure running go dispatcher binary
    read
  fi
}

function run_javac_dispatcher () {
  echo Running Java dispatcher server...
  $JAVAC_DISPATCHER_BIN
  if [ $? -ne 0 ] ; then
    echo ERROR: failure running Java dispatcher binary
    read
  fi
}

function bootstrap () {
  if ! which tmux &> /dev/null ; then
    echo This script needs tmux, run "sudo apt-get install tmux"
    exit 0
  fi
  echo Launching tmux...
  tmux new -s $TMUX_SESSION -n go_dispatcher -d "$0 go_dispatcher" \; \
    new-window -n javac_dispatcher -d "$0 javac_dispatcher" \; \
    new-window -n shipshape -d "$0 shipshape" \; \
    new-window -n about -d "$0 about" \; #\
  tmux attach -t $TMUX_SESSION \; find-window about
  exit 0
}

if [ $# -eq 0 ] ; then
  bootstrap
else
  case "$1" in
      "shipshape")
          run_shipshape
          ;;
      "go_dispatcher")
          run_go_dispatcher
          ;;
      "javac_dispatcher")
          run_javac_dispatcher
          ;;
      "about")
          run_about
          ;;
  esac
fi


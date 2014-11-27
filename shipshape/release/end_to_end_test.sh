#!/bin/bash
# Script that sets up a test repo and then runs the Shipshape CLI
# on that test repo.

set -eux

CONVOY_URL='container.cloud.google.com'
LOCAL_WORKSPACE='/tmp/shipshape-tests'
LOG_FILE='end_to_end_test.log'
REPO=$CONVOY_URL/_b_shipshape_registry
CONTAINERS=(
  //shipshape/release/base
  //shipshape/release
  //shipshape/androidlint_analyzer/release
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
  REPO=""
  done
fi

# Set up log file
touch $LOG_FILE
rm $LOG_FILE
echo ">>> Detailed output will appear in $LOG_FILE"

# Get access token for the convoy docker registry
gcloud preview docker --server=$CONVOY_URL --authorize_only

# Create a test repository to run analysis on
mkdir -p $LOCAL_WORKSPACE
echo "this is not javascript" > $LOCAL_WORKSPACE/test.js

# Run CLI over the new repo
echo "---- Running CLI over test repo" &>> $LOG_FILE
../../campfire-out/bin/shipshape/cli/shipshape --tag=$TAG --repo=$REPO --categories='PostMessage,JSHint' --try_local=$IS_LOCAL_RUN $LOCAL_WORKSPACE >> $LOG_FILE
echo "done."

# Quick sanity checks of output.
JSHINT_COUNT=$(grep -c JSHint $LOG_FILE)
POSTMESSAGE_COUNT=$(grep -c PostMessage $LOG_FILE)
FAILURE_COUNT=$(grep -ci Failure $LOG_FILE)
[[ $JSHINT_COUNT == 8 ]] || { echo "Wrong number of JSHint results, expected 8, found $JSHINT_COUNT" 1>&2 ; exit 1; }
[[ $POSTMESSAGE_COUNT == 1 ]] || { echo "Wrong number of PostMessage results, expected 1, found $POSTMESSAGE_COUNT" 1>&2 ; exit 1; }
[[ $FAILURE_COUNT == 0 ]] || { echo "Some analyses failed; please check $LOG_FILE" 1>&2 ; exit 1; }
echo "Success! Analyzer produced expected number of results. Full output in $LOG_FILE"

exit 0

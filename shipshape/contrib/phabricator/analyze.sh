#!/bin/bash -e
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
# TODO(jvg): remove when gcloud sdk releases fix for refresh token
# authorize docker for use by the shipshape CLI (it pulls down the service)
gcloud auth print-access-token
gcloud preview docker --server=container.cloud.google.com --authorize_only
/shipshape/cli/shipshape --categories="$SHIPSHAPE_CATEGORIES" \
  --json_output=$JSON_OUTPUT $REPO_DIR
#TODO(jvg): delete patch branch

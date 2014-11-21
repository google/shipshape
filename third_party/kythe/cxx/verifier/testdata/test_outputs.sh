#!/bin/bash
# This script checks that the verifier produces the correct debug
# output given pre-baked indexer output. Each test case should have
# a .bin (the result of redirecting the indexer's output to a file),
# a .dot (the correct result from running the verifier using that
# output with the --graphviz flag) and a .json (ditto, but with
# --json). If the output matches exactly for each test case, this
# script returns zero.
HAD_ERRORS=0
VERIFIER=campfire-out/bin/third_party/kythe/cxx/verifier/verifier
BASEDIR=third_party/kythe/cxx/verifier/testdata
function one_case {
  cat $1.bin | ${VERIFIER} --graphviz | diff $1.dot -
  DOT_RESULTS=( ${PIPESTATUS[1]} ${PIPESTATUS[2]} )
  cat $1.bin | ${VERIFIER} --json | diff $1.json -
  JSON_RESULTS=( ${PIPESTATUS[1]} ${PIPESTATUS[2]} )
  if [ ${DOT_RESULTS[0]} -ne 0 ]; then
    echo "[ FAILED VERIFY --GRAPHVIZ: $1 ]"
    HAD_ERRORS=1
  elif [ ${DOT_RESULTS[1]} -ne 0 ]; then
    echo "[ WRONG VERIFY --GRAPHVIZ: $1 ]"
    HAD_ERRORS=1
  elif [ ${JSON_RESULTS[0]} -ne 0 ]; then
    echo "[ FAILED VERIFY --JSON: $1 ]"
    HAD_ERRORS=1
  elif [ ${JSON_RESULTS[1]} -ne 0 ]; then
    echo "[ WRONG VERIFY --JSON: $1 ]"
    HAD_ERRORS=1
  else
    echo "[ OK: $1 ]"
  fi
}

one_case "${BASEDIR}/just_file_node"

exit ${HAD_ERRORS}

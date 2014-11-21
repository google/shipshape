#!/bin/bash
# This script regenerates the flex lexer and bison parser used by the verifier.
# It expects Bison to live at ~/bison/bin/bison. The versions used for the
# artifacts in the repository are bison 3.0 and flex 2.5.35.
set -e
BISON=~/bison/bin/bison
FLEX=flex
SCRIPT=$(readlink -f "$0")
SCRIPT_PATH=$(dirname "$SCRIPT")
pushd ${SCRIPT_PATH}
rm -f assertions.tab.cc assertions.tab.hh location.hh position.hh stack.hh lex.yy.c lex.yy.cc
${FLEX} assertions.ll
mv lex.yy.c lex.yy.cc
${BISON} assertions.yy
popd

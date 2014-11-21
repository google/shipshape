#!/bin/bash
# This script is run by docker when the docker container receives a run
# instruction. It starts the android_lint_service and stores the output to a log
# file.

/usr/sbin/sshd
./android_lint_service >& /tmp/android_lint.log

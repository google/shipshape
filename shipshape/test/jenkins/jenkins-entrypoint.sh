#! /bin/bash

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

set -e

# Add shipshape to PATH so we can run the Jenkins plugin properly.
export PATH="$PATH:/opt/google/shipshape"

# Add JNLP port initialization script to set a fixed JNLP port.
export JNLP_PORT="50000"

if [ ! -d "${JENKINS_HOME}/init.groovy.d" ]; then
  mkdir "${JENKINS_HOME}/init.groovy.d"
fi

export JNLP_PORT_SCRIPT="${JENKINS_HOME}/init.groovy.d/fixed-jnlp-port.groovy"
if [ ! -e "$JNLP_PORT_SCRIPT" ]; then
  echo "Thread.start { sleep 12000" > "$JNLP_PORT_SCRIPT"
  echo "println \"Setting JNLP port to ${JNLP_PORT}\"" >> "$JNLP_PORT_SCRIPT"
  echo "jenkins.model.Jenkins.instance.setSlaveAgentPort(${JNLP_PORT}) }" >> "$JNLP_PORT_SCRIPT"
fi

exec java -jar /usr/share/jenkins/jenkins.war "$@"

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

declare -xr LOCAL_WORKSPACE="/tmp/shipshape-tests"

#######################################
# Creates a test repository.
# Globals:
#   LOCAL_WORKSPACE
# Arguments:
#   None
# Return:
#   None
#######################################
create_test_repo() {
  echo "INFO: Creating test repo at $LOCAL_WORKSPACE"
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

create_test_repo

./shipshape --categories='PostMessage' --stderrthreshold=INFO "$LOCAL_WORKSPACE"

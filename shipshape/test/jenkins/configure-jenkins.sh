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

# Helper script to install plugins and create test job in the Jenkins testing
# instance.

set -e

curl http://localhost:8080/jnlpJars/jenkins-cli.jar -o jenkins-cli.jar

CLI="java -jar jenkins-cli.jar -s http://localhost:8080/ -noKeyAuth"

$CLI install-plugin \
        ../../jenkins_plugin/target/shipshape-plugin.hpi

$CLI install-plugin credentials
$CLI install-plugin scm-api
$CLI install-plugin git-client
$CLI install-plugin git -restart

sleep 20s

$CLI create-job JUnit <<EOF
<project>
  <actions/>
  <description/>
  <keepDependencies>false</keepDependencies>
  <properties/>
  <scm class="hudson.plugins.git.GitSCM" plugin="git@2.3.5">
    <configVersion>2</configVersion>
    <userRemoteConfigs>
      <hudson.plugins.git.UserRemoteConfig>
        <url>https://github.com/junit-team/junit.git</url>
      </hudson.plugins.git.UserRemoteConfig>
    </userRemoteConfigs>
    <branches>
      <hudson.plugins.git.BranchSpec>
        <name>*/master</name>
      </hudson.plugins.git.BranchSpec>
    </branches>
    <doGenerateSubmoduleConfigurations>false</doGenerateSubmoduleConfigurations>
    <submoduleCfg class="list"/>
    <extensions/>
  </scm>
  <canRoam>true</canRoam>
  <disabled>false</disabled>
  <blockBuildWhenDownstreamBuilding>false</blockBuildWhenDownstreamBuilding>
  <blockBuildWhenUpstreamBuilding>false</blockBuildWhenUpstreamBuilding>
  <triggers/>
  <concurrentBuild>false</concurrentBuild>
  <builders>
    <com.google.shipshape.jenkins.AnalysisRunner plugin="shipshape-plugin@0.37">
      <categories>ExtendJ</categories>
      <command>shipshape</command>
      <analyzerImages>localhost:5000/extendj_shipshape/extendj:latest</analyzerImages>
      <verbose>true</verbose>
      <socket>unix:///var/run/docker.sock</socket>
      <stage>PRE_BUILD</stage>
    </com.google.shipshape.jenkins.AnalysisRunner>
  </builders>
  <publishers/>
  <buildWrappers/>
</project>
EOF

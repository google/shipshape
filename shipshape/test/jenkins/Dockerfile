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

#### The following part is taken from shipshape/test/dind/docker/Dockerfile:
FROM gcr.io/_b_dev_containers/cloud-dev-java:prod

# Let's start with some basic stuff.
RUN apt-get update -qq && apt-get install -qqy \
    apt-transport-https \
    ca-certificates \
    curl \
    lxc \
    iptables

# Install Docker from Docker Inc. repositories.
RUN curl -sSL https://get.docker.com/ubuntu/ | sh

# Define additional metadata for our image.
VOLUME /var/lib/docker

#### The rest of this Dockerfile is Jenkins related.

# Install some programs needed by Jenkins.
RUN apt-get install -qqy wget git zip

# Clean up after Apt.
RUN rm -rf /var/lib/apt/lists/*

# Set Jenkins home directory and make it a docker volume.
ENV JENKINS_HOME /var/jenkins_home
VOLUME /var/jenkins_home

# Jenkins version and SHA hash.
ENV JENKINS_VERSION 1.619
ENV JENKINS_SHA 2fce08aaba46cde57398fa484069ab6b95520b7e

RUN mkdir -p /usr/share/jenkins

# Fetch Jenkins.
RUN curl -fL "http://mirrors.jenkins-ci.org/war/${JENKINS_VERSION}/jenkins.war" \
    -o /usr/share/jenkins/jenkins.war

# Check the integrity of the downloaded Jenkins War.
RUN echo "${JENKINS_SHA} /usr/share/jenkins/jenkins.war" | sha1sum -c -

# Web port.
EXPOSE 8080

# Port needed for Jenkins CLI.
EXPOSE 50000

COPY jenkins-entrypoint.sh /usr/local/bin/jenkins-entrypoint.sh

# Copy shipshape binary.
RUN mkdir -p /opt/google/shipshape
COPY shipshape /opt/google/shipshape/shipshape
RUN chmod a+x /opt/google/shipshape/shipshape

#### TODO(joqvist): running Jenkins as root, will this cause problems?
#### joqvist: replace entrypoint script by the ONRUN thing to make it work with cloud-dev-java
ENV ONRUN "${ONRUN}" "/usr/local/bin/jenkins-entrypoint.sh"

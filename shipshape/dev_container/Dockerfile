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

# This docker image is for development of Shipshape istelf. It provides a
# ready-made environment in which Shipshape can be built and tested.
#
# Shipshape has the following build and test dependencies:
#
#  - Docker - Needed by Shipshape
#  - JDK version 8 - Needed by Bazel
#  - Bazel - Needed by Shipshape for building/testing
#  - Bison
#  - Clang
#  - Flex
#  - Go
#  - Android lint - Needed by Shipshape for testing
#
# Dependencies are listed at https://github.com/google/shipshape.
#

# -- Docker --
# This is a dind container and provides us with Docker
FROM beta.gcr.io/dev-con/cloud-dev-java:prod

# Make sure image is up to date
RUN apt-get -qq update && \
    apt-get -qq install --no-install-recommends \
        bison \
        clang \
        flex \
        g++ \ 
        openjdk-8-jdk \
        openjdk-8-source \
        pkg-config \
        unzip \
        zip \
        zlib1g-dev 

# -- Go --
RUN curl -L -s http://golang.org/dl/go1.5.1.linux-amd64.tar.gz | tar -zx -C /usr/local
ENV PATH $PATH:/usr/local/go/bin
ENV GOPATH /go
ENV GOROOT /usr/local/go

# -- Bazel --
RUN wget -nv -O /tmp/bazel-installer.sh https://github.com/bazelbuild/bazel/releases/download/0.1.0/bazel-0.1.0-installer-linux-x86_64.sh && \
    bash /tmp/bazel-installer.sh && \
    rm /tmp/bazel-installer.sh

# -- Android lint --
ENV PATH /android-sdk-linux/platform-tools:/android-sdk-linux/tools:$PATH
# The latter half of this command is a workaround for a failure in the "android
# update" command: https://github.com/google/shipshape/issues/27. The update
# fails to clobber the /tools directory for some reason, so we have to do the
# clobbering for it.
RUN wget -nv -O - http://dl.google.com/android/android-sdk_r23-linux.tgz | tar -zx && \
    echo "y" | android -s update sdk --no-ui --filter platform-tool && \
    echo "y" | android -s update sdk --no-ui --filter tools && \
    ! { stat -t /android-sdk-linux/temp/tools_*-linux.zip; } || \
    { rm -rf /android-sdk-linux/tools && \
      unzip -qq /android-sdk-linux/temp/tools_*-linux.zip -d /android-sdk-linux && \
      rm -rf /android-sdk-linux/temp; }

# startup.sh doesn't actually do anything; this is some kind of Docker-in-Docker
# black magic.
ADD shipshape/dev_container/startup.sh /startup.sh
ENV ONRUN ${ONRUN} "/startup.sh"

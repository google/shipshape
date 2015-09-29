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

# This docker image is for testing of Shipshape. It provides a ready-made
# environment in which Shipshape can be built and tested.
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
FROM b.gcr.io/dev_containers/cloud-dev-java:prod

# Make sure image is up to date
RUN apt-get update -qq && apt-get upgrade -qq && \
   apt-get install -qq --no-install-recommends

# -- JDK 8 ---
RUN apt-get install -qq --no-install-recommends openjdk-8-jdk openjdk-8-source

# -- Go --
RUN curl -L -s http://golang.org/dl/go1.5.1.linux-amd64.tar.gz | tar -zx -C /usr/local
ENV PATH $PATH:/usr/local/go/bin
ENV GOPATH /go
ENV GOROOT /usr/local/go

# -- Bison, Clang, and Flex --
RUN apt-get install -qq --no-install-recommends bison clang flex

# -- Shipshape --
RUN git clone --depth 1 https://github.com/google/shipshape.git

# -- Bazel --
RUN apt-get install -qq --no-install-recommends pkg-config zip g++ zlib1g-dev unzip
RUN wget -nv -O /tmp/bazel-installer.sh https://github.com/bazelbuild/bazel/releases/download/0.1.0/bazel-0.1.0-installer-linux-x86_64.sh
RUN bash /tmp/bazel-installer.sh

# -- Android lint --
RUN wget -nv -O - http://dl.google.com/android/android-sdk_r23-linux.tgz | tar xzf -
ENV PATH /android-sdk-linux/tools:$PATH
RUN echo "y" | android -s update sdk --no-ui --filter platform-tool
RUN echo "y" | android -s update sdk --no-ui --filter tools
# This is a workaround for a failure in the "android update" command:
# https://github.com/google/shipshape/issues/27
# The update fails to clobber the /tools directory for some reason, so
# we have to do the clobbering for it.
RUN rm -rf /android-sdk-linux/tools
RUN unzip -qq /android-sdk-linux/temp/tools_*-linux.zip -d /android-sdk-linux
ENV PATH /android-sdk-linux/platform-tools:$PATH

# Make sure Bazel is setup (and android lint)
WORKDIR /shipshape
RUN ./setup_bazel.sh

ADD startup.sh /startup.sh

# The underlying dind container requires this kind of start up for its dind support
ENV ONRUN ${ONRUN} "/startup.sh"

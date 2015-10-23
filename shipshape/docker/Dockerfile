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

FROM beta.gcr.io/dev-con/cloud-dev-java:prod

# Make sure instance is up to date
RUN apt-get update && apt-get upgrade -y  && \
   apt-get install -y  -qq --no-install-recommends \
# utilities
  sudo curl \
# Packages needed for jshint
#  npm \
  nodejs-legacy moreutils \
# Packages needed for pylint
  pylint=1.3.1-3

# Setup jshint
RUN curl -L https://www.npmjs.org/install.sh | sponge | clean=no   sh
RUN npm install -g jshint

# Checkstyle
ADD third_party/checkstyle/checkstyle-6.11.2-all.jar /usr/local/bin/checkstyle-6.11.2-all.jar

# Install Go, needed for the go vet analyzer
RUN curl -L -s http://golang.org/dl/go1.3.linux-amd64.tar.gz | tar -zx -C /usr/local
ENV PATH $PATH:/usr/local/go/bin
ENV GOPATH /go
ENV GOROOT /usr/local/go

# Set up Shipshape
ADD shipshape/java/com/google/shipshape/service/java_dispatcher_deploy.jar java_dispatcher.jar
ADD shipshape/java/com/google/shipshape/service/javac_dispatcher_deploy.jar javac_dispatcher.jar
ADD shipshape/service/go_dispatcher /go_dispatcher
ADD shipshape/service/shipshape /shipshape
ADD shipshape/docker/endpoint.sh /endpoint.sh

# Expose ports for dispatchers/analyzers
# 10007 - shipshape service
# Dispatcher ports exposed for testing
# 10005 - go dispatcher
# 10006 - javac dispatcher
# 10008 - java dispatcher
EXPOSE 10005 10006 10007 10008

ENTRYPOINT ["/endpoint.sh"]


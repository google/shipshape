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

FROM gcr.io/_b_dev_containers/cloud-dev-java:prod

# Make sure instance is up to date
RUN apt-get update && apt-get upgrade -y  && \
   apt-get install -y  -qq --no-install-recommends

# Add needed packages
RUN apt-get install -y -qq --no-install-recommends

# Install Go
RUN curl -L -s http://golang.org/dl/go1.4.2.linux-amd64.tar.gz | tar -zx -C /usr/local
ENV PATH $PATH:/usr/local/go/bin
ENV GOPATH /go
ENV GOROOT /usr/local/go

# Copy files needed for dind test
ADD shipshape/test/dind/docker/endpoint.sh /endpoint.sh
ADD shipshape/cli/shipshape /shipshape

ENV ONRUN ${ONRUN} "/endpoint.sh"

EXPOSE 10022


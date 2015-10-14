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
# environment in which Shipshape's source code is mounted and ready to be
# built.

FROM beta.gcr.io/shipshape_releases/dev_container:prod

# -- Shipshape --
RUN git clone --depth 1 https://github.com/google/shipshape.git
WORKDIR /shipshape
RUN ./configure

ADD startup.sh /startup.sh

# The underlying dind container requires this kind of start up for its dind support
ENV ONRUN ${ONRUN} "/startup.sh"

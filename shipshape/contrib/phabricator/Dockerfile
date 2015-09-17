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

FROM gcr.io/_b_dev_containers/cloud-dev-nodejs:prod

RUN apt-get update && apt-get upgrade -y

RUN apt-get install -y php5 php5-curl
RUN mkdir /arc
RUN cd /arc && git clone https://github.com/phacility/libphutil.git
RUN cd /arc && git clone https://github.com/phacility/arcanist.git

ENV PATH /arc/arcanist/bin:$PATH
ADD shipshape/contrib/phabricator/async.js /shipshape/async.js
ADD shipshape/contrib/phabricator/conduit.js /shipshape/conduit.js
ADD shipshape/contrib/phabricator/server.js /shipshape/server.js
ADD shipshape/contrib/phabricator/analyze.sh /shipshape/analyze.sh
ADD shipshape/contrib/phabricator/launch.sh /shipshape/launch.sh
ADD shipshape /shipshape/cli/shipshape

RUN cd /shipshape/ && npm install winston

# Support Gerrit
ADD shipshape/contrib/phabricator/gitconfig /etc/gitconfig
ADD shipshape/contrib/phabricator/git-credential-gerrit.sh /google-cloud-sdk/bin/git-credential-gerrit.sh
RUN chmod +x /google-cloud-sdk/bin/git-credential-gerrit.sh

CMD [ "/shipshape/launch.sh"]

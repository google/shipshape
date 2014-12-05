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

# This builds off of the production version of shipshape base.
# If you are testing changes that need a different version of
# base, you will need to (temporarily) modify this tag.
# There is no way to say "whatever tag I am using, use the same one."
FROM container.cloud.google.com/_b_shipshape_registry/base:prod

# Set up Shipshape
ADD javac_dispatcher_deploy.jar javac_dispatcher.jar
ADD go_dispatcher /go_dispatcher
ADD shipshape /shipshape
ADD endpoint.sh /endpoint.sh
EXPOSE 10005 10006 10007
ENTRYPOINT ["/endpoint.sh"]


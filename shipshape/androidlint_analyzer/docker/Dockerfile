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

FROM debian:wheezy

# Make sure all package lists are up-to-date
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y -q  --no-install-recommends \
        sudo openjdk-6-jre openjdk-6-jdk unzip wget && \
    apt-get clean

# Install all the dependencies that the AndroidLint analyzer will need.
# Get the jdk, then android.
# Update the necessary packages one at a time so we can accept each license as needed
RUN wget http://dl.google.com/android/android-sdk_r23-linux.tgz
RUN tar xvf android-sdk_r23-linux.tgz
ENV PATH /android-sdk-linux/tools:$PATH
RUN echo "y" | android update sdk --no-ui --filter platform-tool
RUN echo "y" | android update sdk --no-ui --filter tools
# This is a workaround for a failure in the "android update" command:
# https://github.com/google/shipshape/issues/27
# The update fails to clobber the /tools directory for some reason, so
# we have to do the clobbering for it.
RUN rm -rf /android-sdk-linux/tools
RUN unzip /android-sdk-linux/temp/tools_*-linux.zip -d /android-sdk-linux
ENV PATH /android-sdk-linux/platform-tools:$PATH

# Set up AndroidLintAnalzer
# Add the binary that we'll run in the endpoint script.
ADD shipshape/androidlint_analyzer/androidlint/android_lint_service /android_lint_service
ADD shipshape/androidlint_analyzer/docker/endpoint.sh /endpoint.sh
# 10005 is the port that the shipshape
# service will expect to see a Shipshape Analyzer at.
EXPOSE 10005
# Start the endpoint script.
ENTRYPOINT ["/endpoint.sh"]

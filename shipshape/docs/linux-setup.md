<!--
// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
-->

# Installing shipshape on Linux

Shipshape has been tested on Ubuntu (>=14.04) and Debian unstable, but should work on other Linux distributions.

## Install docker

`apt-get install docker-engine` works for most machines, but [complete
  instructions](https://docs.docker.com/installation) are available.

Make sure you can [run docker without sudo](https://docs.docker.com/articles/basics) by adding your user to the docker group. After you do this, log out of your terminal and log in again.

    $ sudo usermod -a -G docker $USER

## Install shipshape

    $ wget https://storage.googleapis.com/shipshape-cli/shipshape
    $ chmod 555 shipshape
    $ sudo mv shipshape /usr/bin/

## Run it!

That's it! You can now run shipshape with

    $ shipshape <directory>

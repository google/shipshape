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

# Running Shipshape

First, did you install shipshape? You can either do a [Linux
install](https://github.com/google/shipshape/blob/master/shipshape/docs/linux-setup.md)
or [use
GCE](https://github.com/google/shipshape/blob/master/shipshape/docs/gce-setup.md).

Navigate to a directory you'd like to run shipshape on. If you don't have one,
we have a test repo that has some known bugs.

    $ git clone https://github.com/google/shipshape-demo

Run the command line tool. The first time this is run, it's going to be slow as
it needs to download the latest docker image with the analyzers.

    $ shipshape . # This will take a minute.
    $ shipshape . # This will only be a few seconds now.

Get the list of categories

    $ shipshape --show_categories

Let's create a .shipshape file that only runs PyLint and go vet by default.
Please notice that this is a yaml file, and spacing is important! The structure is
defined by
[shipshape_config.proto](https://github.com/google/shipshape/blob/master/shipshape/proto/shipshape_config.proto)

    $ cat > .shipshape <<EOF
    events:
      - event: default
        categories:
          - go vet
          - PyLint
    EOF

Let's also add a pylintrc file

    $ cat > pylintrc <<EOF
    [MASTER]
    errors-only=yes
    EOF

Now when we run, our preferred settings are used

    $ shipshape .

But we can still override them

    $ shipshape --categories="JSHint" .

We can also try out using one of the [external
analyzers](https://github.com/google/shipshape#contributed-analyzers)

    $ shipshape --analyzer_images="gcr.io/shipshape_releases/android_lint:prod"

Let's add that to our shipshape file too. We can also add multiple events, if we
want to have different results when we run the tool in different ways.

    $ cat > .shipshape <<EOF
    global:
      images:
        - gcr.io/shipshape_releases/android_lint:prod
    events:
      - event: default
        categories:
          - go vet
          - PyLint
          - AndroidLint
    events:
      - event: IDE
        categories:
          - go vet
          - PyLint
    EOF


And it all still works

    $ shipshape .
    $ shipshape --event=IDE .


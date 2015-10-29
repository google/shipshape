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

First, did you [install shipshape](https://github.com/google/shipshape/blob/master/shipshape/docs/linux-setup.md)?

Navigate to a directory you'd like to run shipshape on. If you don't have one,
we have a test repo that has some known bugs.

    git clone https://github.com/google/shipshape-demo

Run the command line tool. The first time this is run, it's going to be slow as
it needs to download the latest docker image with the analyzers.

    shipshape . # This will take a minute.
    shipshape . # This will only be a few seconds now.

When these have completed you should see output that looks something like:

```
shipshape-demo/js-tests/sample.js
Line 18, Col 22 [JSHint]
	Missing semicolon.
Line 19, Col 24 [JSHint]
	Use '===' to compare with 'null'.
...
```

To get the list of categories run:

    shipshape --show_categories

Let's create a .shipshape file that only runs PyLint and go vet by default.
Please notice that this is a yaml file, and spacing is important! The structure is
defined by
[shipshape_config.proto](https://github.com/google/shipshape/blob/master/shipshape/proto/shipshape_config.proto)

    cat > .shipshape <<EOF
    events:
      - event: default
        categories:
          - go vet
          - PyLint
    EOF

Let's also add a pylintrc file

    cat > pylintrc <<EOF
    [MASTER]
    errors-only=yes
    EOF

Now when we run, our preferred settings are used as can be seen by running again:

    shipshape .

But we can still override them

    shipshape --categories="JSHint" .

We can also try out using one of the [external
analyzers](https://github.com/google/shipshape#contributed-analyzers). Like
before, this will take a minute the first time it is run, since it is pulling a
new image down. Let's first find out what categories are available with the
new analyzer.

    shipshape --analyzer_images="joqvist/extendj_shipshape" --show_categories

And now let's run it. Notice that we now have to specify the category, since we
set up the .shipshape file to only run go vet and PyLint.

    shipshape --analyzer_images="joqvist/extendj_shipshape" --categories="ExtendJ" .

Let's add that to our shipshape file too. We can also add multiple events, if we
want to have different results when we run the tool in different ways.

    cat > .shipshape <<EOF
    global:
      images:
        - joqvist/extendj_shipshape
    events:
      - event: default
        categories:
          - go vet
          - PyLint
          - ExtendJ
      - event: IDE
        categories:
          - go vet
          - PyLint
    EOF


And it all still works

    shipshape .
    shipshape --event=IDE .


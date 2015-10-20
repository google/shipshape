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

# Building shipshape from source

These instructions are for people who are contributing to shipshape or want to
build from source. If you just want to run shipshape, please see [the
documentation for installing
shipshape](https://github.com/google/shipshape/blob/master/shipshape/docs/run-cli.md). If you want to add an analyzer, we
have [simple instructions for adding
analyzers](https://github.com/google/shipshape/blob/master/shipshape/docs/add-an-analyzer.md) that do not require building
shipshape from source .

## Dependencies
Here's all the dependencies you're going to need
to build shipshape from source:

  * [Bazel](http://bazel.io), follow these [installation
  instructions](http://bazel.io/docs/install.html).
  * [Bison](https://www.gnu.org/software/bison/)
  * [Clang](http://llvm.org/releases/download.html)
  * [Docker](https://docs.docker.com/docker/userguide), see above instructions.
  * [Flex](http://flex.sourceforge.net/)
  * [Go](http://golang.org/doc/install)
  * [JDK 8](http://docs.oracle.com/javase/8/docs/technotes/guides/install/install_overview.html)

You can pull a Docker image with all of these dependencies and develop in that:

    ```
    docker pull gcr.io/shipshape_releases/dev_container:prod
    docker run --privileged -it gcr.io/shipshape_releases/dev_container:prod /bin/bash
    ```

Or you can install bison, clang, flex, and go (on Ubuntu >=14.10) using apt:

    ```
    sudo apt-get install bison clang flex golang openjdk-8-jdk openjdk-8-source
    ```

To run tests for Shipshape you also need Android `lint` (part of the
[Android SDK](https://developer.android.com/sdk/index.html)) installed in
your system `PATH`.


## Building our source

    mkdir -p github.com/google && cd github.com/google
    git clone https://github.com/google/shipshape.git
    cd shipshape
    ./configure        # Run initial Shipshape+Bazel setup
    bazel build //...  # Build all Shipshape source

## Running locally
Bazel puts the Shipshape CLI binary in the bazel-bin directory. You can run it
on you directory:

```
./bazel-bin/shipshape/cli/shipshape <Directory>
```

## Run with Local Docker Images

The Shipshape CLI uses released docker images for Shipshape by default. If you
pass `--tag local` to the CLI it will use locally built images instead.

To build and store docker images locally, run:

```
bazel build //shipshape/docker:service
bazel build //shipshape/androidlint_analyzer/docker:android_lint
```

To run with local images:

```
./bazel-bin/shipshape/cli/shipshape --tag=local <Directory>
```

## Testing

For unit tests, run:

```
bazel test //...
```

For the end-to-end test, run:

```
bazel test //shipshape/cli:test_local
```




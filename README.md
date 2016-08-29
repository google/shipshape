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

# Overview of Shipshape #

[![Build Status](https://travis-ci.org/google/shipshape.svg?branch=master)](https://travis-ci.org/google/shipshape)

Shipshape is a static program analysis platform that allows custom analyzers to
plug in through a common interface. Shipshape is packaged in a docker image.
When that image is run, a Shipshape analyzer service starts up and processes
analysis requests. Structured analysis results are generated. Shipshape can be
run as a command-line interface, or as a Jenkins plugin. The requirements to run
are that you are running Linux with docker installed and the source code you want
to analyze available on disk.

The source code for Shipshape is located in the "shipshape" directory.
Third-party libraries used by Shipshape are all in the "third_party" directory.

## Download and Run Shipshape

Shipshape has been tested on Ubuntu (>=14.04) and Debian unstable, but should work on other Linux distributions.

Shipshape requires [Docker](https://docs.docker.com/docker/userguide/) to run.

[Install instructions for Linux](shipshape/docs/linux-setup.md).

[Install instructions for GCE](shipshape/docs/gce-setup.md).

Once you've installed it, running is easy!

    $ shipshape <Directory>

For examples for how to use it, [see our documentation](shipshape/docs/run-cli.md).

### Building from source

Shipshape uses the [Bazel build tool](http://bazel.io/docs/install.html). Once you have Docker and Bazel installed, you can build Shipshape with:

    $ ./configure
    $ bazel build //...

The binary will be saved in `bazel-bin/shipshape/cli/shipshape`.

## Analyzers

The following analyzers are bundled with Shipshape:

* [go vet](https://godoc.org/github.com/golang/go/src/cmd/vet)
* [JSHint](http://www.jshint.com/)
* [PyLint](http://www.pylint.org/ )
* [Error Prone](https://github.com/google/error-prone) (category: `ErrorProne`) ***[Under construction: [Issue #104](https://github.com/google/shipshape/issues/104)]***

### Contributed analyzers

The following analyzers were contributed by external developers:

* [AndroidLint](http://tools.android.com/tips/lint). Image: `gcr.io/shipshape_releases/android_lint:prod`
* [CTADetector](http://mir.cs.illinois.edu/~yulin2/CTADetector) - Yu Lin (University of Illinois at Urbana-Champaign). Image: `yulin2/ctadetector`
* [ExtendJ](https://github.com/google/simplecfg) - Jesper Öqvist (Lund University). Image: `joqvist/extendj_shipshape`

### Add a new analyzer

See our [documentation](shipshape/docs/add-an-analyzer.md) on how to create more analyzers of your own.
We also have [a complete example](shipshape/androidlint_analyzer/README.md).

## Contributing to shipshape

To contribute to shipshape, first [read our contribution guidelines](CONTRIBUTING.md) and then
make sure you can [build and run shipshape from source](shipshape/docs/dev-setup.md).

## Running the Jenkins Plugin #

Instructions are located in `shipshape/jenkins_plugin/README.md`.

## Package Structure of Shipshape #

**analyzers** -- implementation for several simple analyzers run by the
  go_dispatcher. The canonical simplest analyzer is in analyzers/postmessage

**androidlint_analyzer** -- implementation for AndroidLint packaged as a complete
  Shipshape analyzer service, using libraries from the service package

**api** -- go API used by analyzers running under the go_dispatcher

**cli** -- code for the CLI that pulls down a Shipshape service, starts it running
  on a specified directory, and outputs analysis results

**docker** -- Dockerfiles for the various docker packages produced by Shipshape

**java** -- code for a javac dispatcher analyzer service that runs analyzers that
  build off of javac

**jenkins_plugin** -- code for the jenkins plugin that runs Shipshape

**proto** -- the protocol buffer APIs for writing new analyzers. Shipshape analyzers
  are services that implement the rpcs listed in the ShipshapeService interface
  in proto/shipshape_rpc.proto. Analyzers produce structured output in the form
  of Note messages, defined in proto/note.proto

**service** -- core Shipshape code.
  go_dispatcher -- dispatching Shipshape analyzer service for the go language.
    calls out to analyzers in the analyzer package.
  shipshape -- main shipshape service loop
  driver -- controller for calling out to all passed in analyzer services
    (including the go_dispatcher and the javac_analyzer)
  config -- processes .shipshape config files to determine which analyzers run

**test** -- manual integration tests to simplify the process of running Shipshape
  locally on test input, useful when developing new analyzer services

**util** -- various go utilities that simplify Shipshape code, e.g. for working with
  slices, execing docker commands, or writing tests

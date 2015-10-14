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

# Download and Run ShipShape #
Install dependencies, download and run Shipshape.

## System Requirements ##
* Linux: tested on Ubuntu (>=14.04) and Debian unstable, but should work on other Linux distributions.

## Dependencies ##
Shipshape requires [Docker](https://docs.docker.com/docker/userguide/) to run.
  
  `apt-get install docker-engine` works for most machines, but [complete
  instructions](https://docs.docker.com/installation) are available.
  
  Make sure you can [run docker without sudo](https://docs.docker.com/articles/basics) by adding your user to the docker group. After you do this, log out of your terminal and log in again.

         $ sudo usermod -G docker $USER    # Group may have to be created

## Download and Run ##

Download the CLI from http://storage.googleapis.com/shipshape-cli/shipshape

Run it!

```
$ ./shipshape <Directory>
```

For examples for how to use it, [see our
documentation](https://github.com/google/shipshape/blob/master/shipshape/docs/run-cli.md).

## Analyzers ##

The following analyzers are bundled with Shipshape:

* [Error Prone](https://github.com/google/error-prone) (category: `ErrorProne`) ***[Currently broken: [Issue #104](https://github.com/google/shipshape/issues/104)]***
* [go vet](https://godoc.org/golang.org/x/tools/cmd/vet)
* [JSHint](http://www.jshint.com/)
* [PyLint](http://www.pylint.org/ )

### Contributed analyzers ###

The following analyzers were contributed by external developers:

* [AndroidLint](http://tools.android.com/tips/lint). Image: `gcr.io/shipshape_releases/android_lint:prod`.
* [CTADetector](http://mir.cs.illinois.edu/~yulin2/CTADetector) - Yu Lin (University of Illinois at Urbana-Champaign). Image: `yulin2/ctadetector`.
* [ExtendJ](https://github.com/google/simplecfg) - Jesper Öqvist (Lund University). Image: `joqvist/extendj_shipshape`.

### Add a new analyzer

See our
[documentation](https://github.com/google/shipshape/blob/master/shipshape/docs/add-an-analyzer.md) on how to create more analyzers of your own. We also have [a complete example](https://github.com/google/shipshape/tree/master/shipshape/androidlint_analyzer/README.md).

# Run Shipshape from Source #

## System Requirements ##
* Linux: tested on Ubuntu (>=14.04) and Debian unstable, but should work on other Linux distributions.

## Dependencies ##
To build Shipshape you need the following tools:

* [Bazel](http://bazel.io), follow these [installation
  instructions](http://bazel.io/docs/install.html).
* [Bison](https://www.gnu.org/software/bison/)
* [Clang](http://llvm.org/releases/download.html)
* [Docker](https://docs.docker.com/docker/userguide), see above instructions.
* [Flex](http://flex.sourceforge.net/)
* [Go](http://golang.org/doc/install)
* [JDK 8](http://docs.oracle.com/javase/8/docs/technotes/guides/install/install_overview.html)

You can pull a Docker image with all of these dependencies:
```
$ docker pull gcr.io/shipshape_releases/dev_container:prod
$ docker run --privileged -it gcr.io/shipshape_releases/dev_container:prod /bin/bash
```

Or you can install bison, clang, flex, and go (on Ubuntu >=14.10) using apt:
```
$ sudo apt-get install bison clang flex golang openjdk-8-jdk openjdk-8-source
```

To run tests for Shipshape you also need Android `lint` (part of the [Android SDK](https://developer.android.com/sdk/index.html)) installed in your system `PATH`.

## Building ##

```
$ ./configure        # Run initial Shipshape+Bazel setup
$ bazel build //...  # Build all Shipshape source
```

## Running ##

Bazel puts the Shipshape CLI binary in the bazel-bin directory. You can run it
on you directory:

```
$ ./bazel-bin/shipshape/cli/shipshape <Directory>
```

### Run with Local Docker Images ###

The Shipshape CLI uses released docker images for Shipshape by default. If you
pass `--tag local` to the CLI it will use locally built images instead.

To build and store docker images locally, run:

```
$ bazel build //shipshape/docker:service
$ bazel build //shipshape/androidlint_analyzer/docker:android_lint
```

To run with local images:

```
$ ./bazel-bin/shipshape/cli/shipshape --tag=local <Directory>
```

## Testing ##

For unit tests, run:

```
$ bazel test //...
```

For the end-to-end test, run:

```
$ bazel test //shipshape/cli:test_local
```

# Running the Jenkins Plugin #

Instructions are located in `shipshape/jenkins_plugin/README.md`.


# Package Structure of Shipshape #

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

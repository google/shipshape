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
Shipshape relies on the following external dependencies:

* [Docker](https://docs.docker.com/docker/userguide/)
  
  Installation instructions: [ubuntu](https://docs.docker.com/installation/ubuntulinux), [debian](https://docs.docker.com/docker/installation/debian/).
  
  Make sure you can [run docker without sudo](https://docs.docker.com/articles/basics) by adding your user to the docker
group and restarting docker:

         $ sudo usermod -G docker $USER    # Group may have to be created
         $ sudo service docker.io restart

## Download and Run ##

Download the CLI from http://storage.googleapis.com/shipshape-cli/shipshape

Run it!

```
$ ./shipshape --categories="go vet,JSHint,PyLint" <Directory>
```

Get help!

```
$ ./shipshape --help
```


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

Install bison, clang, flex, and go (on Ubuntu >=14.10) using apt:

```
$ sudo apt-get install bison clang flex golang openjdk-8-jdk openjdk-8-source
```

To run tests for Shipshape you also need the following tool:

* Android `lint` (part of the [Android SDK](https://developer.android.com/sdk/index.html)), install in your
system `PATH`.

## Building ##

```
$ ./configure        # Run initial Shipshape+Bazel setup
$ bazel build //...  # Build all Shipshape source
```

## Running ##

Bazel puts the Shipshape CLI binary in the bazel-bin directory. You can run it
on you directory:

```
$ ./bazel-bin/shipshape/cli/shipshape --categories="go vet,JSHint,PyLint" <Directory>
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
$ ./shipshape/test/end_to_end_test.sh --tag local
```

## Testing ##

For unit tests, run:

```
$ bazel test //...
```

For the end-to-end test, run:

```
$ ./shipshape/test/end_to_end_test.sh --tag local
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


# Writing an Analyzer #

To write a new analyzer service, you can use the androidlint_analyzer as an example.

**androidlint/analyzer.go** -- implements the analyzer interface. Basically a wrapper
  that calls out to androidlint as subprocess and translates the output into Notes
  (see the proto dir for more information on the Note message).

**androidlint/service.go** -- sets up running android lint as a service. You will want
  to copy this file and update the names to reflect your analyzer.

**androidlint/analyzer_test.go** -- some sample tests of the analyzer.

**androidlint/CAMPFIRE** -- build file for this analyzer. Should copy and update names.

**docker/Dockerfile,endpoint.sh** -- Dockerfile and shell script need to build a docker
  image containing this analyzer. All dependencies needed to run the analyzer should
  be pulled down in the Dockerfile and the image must run a service on port 10005.

**docker/CAMPFIRE** -- build file for creating a docker image. Should copy and update names.

To build and test the android lint analyzer, run:

```
$ bazel build //shipshape/androidlint_analyzer/androidlint/...
$ bazel test //shipshape/androidlint_analyzer/androidlint/...
```

To build the android lint docker image, run:

```
$ bazel build //shipshape/androidlint_analyzer/docker:android_lint
```

Once you have built an image, verify that it shows up in your list of docker images:

```
$ docker images
```

Now, you can run the shipshape CLI with your analyzer added by passing in its category
name via the `--analyzer_images` flag:

```
$ ./bazel-bin/shipshape/cli/shipshape --categories="AndroidLint" \
    --analyzer_images=android_lint:local --tag=local <Directory>
```

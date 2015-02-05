# Overview of Shipshape #

Shipshape is a static program analysis platform that allows custom analyzers to
plug in through a common interface. Shipshape is packaged in a docker image.
When that image is run, a Shipshape analyzer service starts up and processes
analysis requests. Structured analysis results are generated. Shipshape can be
run as a command-line interface, or as a Jenkins plugin. The requirements to run
are that you are running Linux with docker installed and the source code you want
to analyze available on disk.

The source code for Shipshape is located in the "shipshape" directory. Third-
party libraries used by Shipshape are all in the "third_party" directory.
Shipshape uses a build tool called "campfire"; the source code for this build
tool is located in third_party/buildtools.


# Setup #

See if you have docker installed:

`$ which docker`

If you don't have docker installed:

`$ apt-get install docker.io`


# Building/running CLI entirely locally #

`$ ./test/end_to_end_test.sh local`

This will build the shipshape CLI and all docker containers used locally, and
also run them once on test input. To rerun the CLI on your code with the locally
build docker images:

`$ ./campfire-out/bin/shipshape/cli/shipshape --categories="go vet,JSHint,PyLint" \
      --tag=local <Directory>`


# Building/running CLI with prod docker images #

If you want to pull the latest (released) version of shipshape, you need
"gcloud preview docker". You can get this by running:

`$ gcloud components update preview`

To build shipshape:

`$ ./campfire build shipshape/...`

To run the shipshape CLI (requires access to a GCS bucket, email shipshape@google.com first):

`$ ./campfire-out/bin/shipshape/cli/shipshape --categories="go vet,JSHint,PyLint" <Directory>`


# Running the Jenkins plugin #

Instructions are located in shipshape/jenkins_plugin/README.md


# Package structure of shipshape #

analyzers -- implementation for several simple analyzers run by the
  go_dispatcher. The canonical simplest analyzer is in analyzers/postmessage

androidlint_analyzer -- implementation for AndroidLint packaged as a complete
  Shipshape analyzer service, using libraries from the service package

api -- go API used by analyzers running under the go_dispatcher

cli -- code for the CLI that pulls down a Shipshape service, starts it running
  on a specified directory, and outputs analysis results

docker -- Dockerfiles for the various docker packages produced by Shipshape

java -- code for a javac dispatcher analyzer service that runs analyzers that
  build off of javac

jenkins_plugin -- code for the jenkins plugin that runs Shipshape

proto -- the protocol buffer APIs for writing new analyzers. Shipshape analyzers
  are services that implement the rpcs listed in the ShipshapeService interface
  in proto/shipshape_rpc.proto. Analyzers produce structured output in the form
  of Note messages, defined in proto/note.proto

service -- core Shipshape code.
  go_dispatcher -- dispatching Shipshape analyzer service for the go language.
    calls out to analyzers in the analyzer package.
  shipshape -- main shipshape service loop
  driver -- controller for calling out to all passed in analyzer services
    (including the go_dispatcher and the javac_analyzer)
  config -- processes .shipshape config files to determine which analyzers run

test -- manual integration tests to simplify the process of running Shipshape 
  locally on test input, useful when developing new analyzer services

test_data -- test data used by unit and integration tests

util -- various go utilities that simplify Shipshape code, e.g. for working with
  slices, execing docker commands, or writing tests


# Writing an analyzer #

To write a new analyzer service, you can use the androidlint_analyzer as an example

androidlint/analyzer.go -- implements the analyzer interface. Basically a wrapper
  that calls out to androidlint as subprocess and translates the output into Notes
  (see the proto dir for more information on the Note message).
androidlint/service.go -- sets up running android lint as a service. You will want
  to copy this file and update the names to reflect your analyzer.
androidlint/CAMPFIRE -- build file for this analyzer. Should copy and update names.
docker/Dockerfile,endpoint.sh -- Dockerfile and shell script need to build a docker
  image containing this analyzer. All dependencies needed to run the analyzer should
  be pulled down in the Dockerfile.
docker/CAMPFIRE -- build file for creating a docker image. Should copy and update names.

You can try building the android lint docker image by running (in root of repo):
    ./campfire package --start_registry=false --docker_tag=local //shipshape/androidlint_analyzer/docker:android_lint

Once you have built an image, verify that it shows up in your list of docker images:
    docker ps

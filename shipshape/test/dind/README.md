# Docker in docker (dind) test #

Tests the Shipshape CLI when running inside a docker container.
The test has three components:

 - a [docker container)(docker/Dockerfile) with shipshape CLI dependencies installed,
 - an [endpoint script](shipshape/test/dind/docker/endpoint.sh) running the CLI with the --insideDocker flag for the
   PostMessage category, and
 - a [test script](shipshape/test/dind/dind_test.sh) that build the docker container and runs it.

The full test is run from the test script:

`$ ./dind_test.sh`

The script builds the container and runs it. This may take some time as it is
pulling down the Shipshape production images from in the docker container.

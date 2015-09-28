# Docker test image #

Creates a docker test image that allows you to build and test Shipshape without
setting up all its dependencies, with the exception of docker which you need to
build the test image.

To build and run the test image, run:

`$ ./docker_test.sh`

The script builds the test image and then starts a container using the image and bash.
That is, it will place you in a bash terminal at the root of the cloned
Shipshape repo inside the container running the test image.

The container will have the latest version of Shipshape and it's dependencies.
To build and test Shipshape, navigate to /shipshape and run bazel:

`$ cd /shipshape && bazel test ...`


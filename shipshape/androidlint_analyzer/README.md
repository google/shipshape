# A complete example of an external analyzer
See our
[documentation](https://github.com/google/shipshape/blob/master/shipshape/docs/add-an-analyzer.md) on how to create more analyzers of your own.

**androidlint/analyzer.go** -- implements the analyzer interface. Basically a wrapper
  that calls out to androidlint as subprocess and translates the output into Notes
  (see the proto dir for more information on the Note message).

**androidlint/service.go** -- sets up running android lint as a service. You will want
  to copy this file and update the names to reflect your analyzer.

**androidlint/analyzer_test.go** -- some sample tests of the analyzer.

**androidlint/BUILD** -- build file for this analyzer. We are using Bazel, but
any build system will work.

**docker/Dockerfile,endpoint.sh** -- Dockerfile and shell script need to build a docker
  image containing this analyzer. All dependencies needed to run the analyzer should
  be pulled down in the Dockerfile and the image must run a service on port 10005.

**docker/BUILD** -- build file for creating a docker image.

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

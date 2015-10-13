# Running Shipshape

Run the command line tool

    ./shipshape .

Get the list of categories

    ./shipshape --show_categories

Let's create a .shipshape file that only runs PyLint and go vet by default.
Please notice that this is a yaml file, and spacing is important! The structure is
defined by
[shipshape_config.proto](https://github.com/google/shipshape/blob/master/shipshape/proto/shipshape_config.proto)

    cat > .shipshape <<EOF
    events:
      - event: default
        categories:
          - go vet
          - Py Lint
    EOF

Let's also add a pylintrc file

    cat > pylintrc <<EOF
    [MASTER]
    errors-only=yes
    EOF

Now when we run, our preferred settings are used

    ./shipshape .

But we can still override them

    ./shipshape --categories="JSHint" .

We can also try out using one of the [external analyzers](TODOTODO)

    ./shipshape --analyzer_images="gcr.io/shipshape_releases/android_lint:prod"

Let's add that to our shipshape file too. We can also add multiple events, if we
want to have different results when we run the tool in different ways.

    cat > .shipshape <<EOF
    global:
      images:
        - gcr.io/shipshape_releases/android_lint:prod
    events:
      - event: default
        categories:
          - go vet
          - Py Lint
          - AndroidLint
    events:
      - event: IDE
        categories:
          - go vet
          - Py Lint
    EOF


And it all still works

    ./shipshape .
    ./shipshape --event=IDE .


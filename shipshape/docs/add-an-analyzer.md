// Copyright 2014 Google Inc. All rights reserved.
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

# Deploy an analyzer using Shipshape

## Dependencies

Let's first make sure you're set up with the necessary tools to build and deploy a shipshape analyzer. You will need:

* Docker
* The Shipshape CLI
* The Shipshape API (currently supporting go and Java)
* Whatever language tools and dependencies your analyzer needs to build and run

The Linux instructions work for Ubuntu 10.14+. Alternately, you can create a GCE instance from our provided setup image, which has everything you need pre-installed. We do not support Mac/Windows at this time, due to a bug in Docker.

For this tutorial, we be creating an analyzer implemented in Go.

### GCE
First, make a Google Compute Engine Project. TODO

Create an instance using the shipshape image. TODO instructions and link to the
image

SSH into your instance TODO

### Linux

If you do not already have it, install docker

    $ sudo apt-get install docker-engine

If you do not already have it, install go by following the [go install instructions](https://golang.org/doc/install)

Get shipshape's go API

    $ go get github.com/google/shipshape/shipshape/api

Install the shipshape CLI

    $ wget http://storage.googleapis.com/shipshape-cli/shipshape
    $ chmod 555 shipshape
    $ sudo mv shipshape /usr/bin/

## Implement your analyzer

Creating an analyzer involves making three things:
1. Creating an analyzer. We recommend implementing our provided API, but this
   can be done a language of your choise.
2. A service that exposes the analyzer as a service. The required API is defined
   by [shipshape_rpc.proto](https://github.com/google/shipshape/blob/master/shipshape/proto/shipshape_rpc.proto). If you utilize the provided library
   and implemented the provided API in step 1, we implement the hard parts for you.
3. A docker image that starts the service and exposes it on port 10005.

### Go

### Create an analyzer
We'll do this by implementing [api.Analyzer](https://github.com/google/shipshape/blob/master/shipshape/api/analyzer.go). The [AndroidLint
analyzer](https://github.com/google/shipshape/blob/master/shipshape/androidlint_analyzer/androidlint/analyzer.go)
is a helpful example

First, implement `Category()`. This is the name of the analyzer, and all results
returned from this analyzer should use this as the name.

```
package myanalyzer

import (
  "github.com/golang/protobuf/proto"

  notepb "github.com/google/shipshape/shipshape/proto/note_proto"
  ctxpb "github.com/google/shipshape/shipshape/proto/shipshape_context_proto"
)

type Analyzer struct{}

func (Analyzer) Category() string { return "HelloWorld" }

```

Implement a simple `Analyze` method that returns a single note

```
func (ala Analyzer) Analyze(ctx *ctxpb.ShipshapeContext) ([]*notepb.Note, error) {
  return []*notepb.Note{
    &notepb.Note{
      Category:    proto.String(ala.Category()),
      Subcategory: proto.String("greetings"),
      Description: proto.String("Hello world, this is a code note"),
      Location: &notepb.Location{
        SourceContext: ctx.SourceContext,
      },
    },
  }
}
```

TODO explain what a note actually is, link to it, explain what shipshape context
is, link to it


### Implement a server for your analyzer
Now, we just need to implement a service that runs on port 10005 and calls to your analyzer. You can use api.Service to help with this.
As an example, see the [AndroidLint service](https://github.com/google/shipshape/blob/master/shipshape/androidlint_analyzer/androidlint/service.go)

```
package main

import (
  "flag"
  "fmt"
  "log"
  "net/http"
  "os"

  "myanalyzer"
  "github.com/google/shipshape/shipshape/api"
  "github.com/google/shipshape/shipshape/util/rpc/server"

  ctxpb "github.com/google/shipshape/shipshape/proto/shipshape_context_proto"
)

var (
  servicePort = flag.Int("port", 10005, "Service port")
)

func main() {
  flag.Parse()

  // The shipshape service will connect to an AnalyzerService
  // at port 10005 in the container. (The service will map this to a different external
  // port at startup so that it doesn't clash with other analyzers.)
  s := server.Service{Name: "AnalyzerService"}
  // Make a new analyzer service. This runs at the "PRE_BUILD" stage, but you can also create analyzer
  // that require build outputs.
  as := api.CreateAnalyzerService([]api.Analyzer{new(myanalyzer.Analyzer)}, ctxpb.Stage_PRE_BUILD)
  if err := s.Register(as); err != nil {
    log.Fatalf("Registering analyzer service failed: %v", err)
  }

  addr := fmt.Sprintf(":%d", *servicePort)
  fmt.Fprintf(os.Stderr, "-- Starting server endpoint at %q\n", addr)
  http.Handle("/", server.Endpoint{&s})

  if err := http.ListenAndServe(addr, nil); err != nil {
    log.Fatalf("Server startup failed: %v", err)
  }
}
```

### Java
Java instructions will be available soon.

## Create a Docker file
* Create a docker file that
    * Adds all dependencies needed to run your analyzer
    * Adds your analyzer
    * Has port 10005 open
    * Starts your service
    https://github.com/google/shipshape/blob/master/shipshape/androidlint_analyzer/docker/DockerFile
* Create an endpoint script that starts the service
https://github.com/google/shipshape/blob/master/shipshape/androidlint_analyzer/docker/endpoint.sh

##  Test your analyzer locally

Build a docker image with the tag "local"
    
    docker build myanalyzer/DockerFile --name=sflwslfms --tag=local

Run the local analyzer

    shipshape --analyzer_image=sdkfjslkfjds --categories=lk;fkdslk directory

## Push it up to gcr.io or docker.io

    docker tag
    docker push sldfjslfjslkdfsldkfm

## Test your public analyzer

   shipshape --analyzer_image=sdkfjslkfjds --categories=lk;fkdslk

Add it to our list!

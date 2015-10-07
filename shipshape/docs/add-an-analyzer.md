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
* The Shipshape API (currently supporting go and Java)
* The Shipshape CLI
* Whatever language tools and dependencies your analyzer needs to build and run

The Linux instructions work for Ubuntu 10.14+. Alternately, you can create a GCE instance from our provided setup image, which has everything you need pre-installed. We do not support Mac/Windows at this time, due to a bug in Docker.

For this tutorial, we be creating an analyzer implemented in Go.

### GCE
First, make a Google Compute Engine Project. TODO

Create an instance using the shipshape image. TODO instructions and link to the
image

SSH into your instance TODO

### Linux

Install docker

    $ sudo apt-get install docker-engine

Install go

    $ wget go stufff here

Get shipshape's go API

    $ go get github.com/google/shipshape/shipshape/api

OR alternately, get the Java API

	TODO: provide the java instructions

Install the shipshape CLI

    $ wget the shipshape CLI
    $ put it in /usr/bin

## Implement your analyzer

### Go

### Implement api.Analyzer
Example:
  https://github.com/google/shipshape/blob/master/shipshape/androidlint_analyzer/androidlint/analyzer.go

Implement a simple ShouldRun method. 

    return true

Implement a simple Analyze method that returns a single note

    return &Note....

### Implement a server for your analyzer
  Example:
  https://github.com/google/shipshape/blob/master/shipshape/androidlint_analyzer/androidlint/service.go

Use api.Service

   example here

### Java
TODO

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

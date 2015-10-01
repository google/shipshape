/*
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Binary shipshape is a command line interface to shipshape.
// It (optionally) pulls a docker container, runs it,
// and runs the analysis service with the specified local
// files and configuration.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"shipshape/cli"
)

var (
	analyzerImages = flag.String("analyzer_images", "", "Full docker path to images of external analyzers to use (comma-separated)")
	build          = flag.String("build", "", "The name of the build system to use to generate compilation units. If empty, will not run the compilation step. Options are maven and go.")
	categories     = flag.String("categories", "", "Categories to trigger (comma-separated). If none are specified, will use the .shipshape configuration file to decide which categories to run.")
	dind           = flag.Bool("inside_docker", false, "True if the CLI is run from inside a docker container")
	event          = flag.String("event", "manual", "The name of the event to use")
	jsonOutput     = flag.String("json_output", "", "When specified, log shipshape results to provided .json file")
	repo           = flag.String("repo", "gcr.io/shipshape_releases", "The name of the docker repo to use")
	stayUp         = flag.Bool("stay_up", true, "True if we should keep the container running, false if we should stop and remove it.")
	tag            = flag.String("tag", "prod", "Tag to use for the analysis service image. If this is local, we will not attempt to pull the image.")
	useLocalKythe  = flag.Bool("local_kythe", false, "True if we should not pull down the kythe image. This is used for testing a new kythe image.")
)

const (
	returnNoFindings = 0
	returnFindings   = 1
	returnError      = 2
)

func main() {
	flag.Parse()

	// Get the file/directory to analyze.
	if len(flag.Args()) != 1 {
		fmt.Println("Usage: shipshape [OPTIONS] <directory>")
		os.Exit(returnError)
	}

	thirdPartyAnalyzers := []string{}
	if *analyzerImages != "" {
		thirdPartyAnalyzers = strings.Split(*analyzerImages, ",")
	}
	cats := []string{}
	if *categories != "" {
		cats = strings.Split(*categories, ",")
	}

	options := cli.Options{
		File:                flag.Arg(0),
		ThirdPartyAnalyzers: thirdPartyAnalyzers,
		Build:               *build,
		TriggerCats:         cats,
		Dind:                *dind,
		Event:               *event,
		JsonOutput:          *jsonOutput,
		Repo:                *repo,
		StayUp:              *stayUp,
		Tag:                 *tag,
		LocalKythe:          *useLocalKythe,
	}

	numResults, err := cli.New(options).Run()
	if err != nil {
		fmt.Printf("Error: %v", err.Error())
		os.Exit(returnError)
	}
	if numResults != 0 {
		os.Exit(returnFindings)
	}
	os.Exit(returnNoFindings)
}

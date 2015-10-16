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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/shipshape/shipshape/cli"
	"github.com/google/shipshape/shipshape/util/defaults"

	notepb "github.com/google/shipshape/shipshape/proto/note_proto"
	rpcpb "github.com/google/shipshape/shipshape/proto/shipshape_rpc_proto"
)

var (
	analyzerImages = flag.String("analyzer_images", "", "Full docker path to images of external analyzers to use (comma-separated)")
	build          = flag.String("build", "", "The name of the build system to use to generate compilation units. If empty, will not run the compilation step. Options are maven and go.")
	categories     = flag.String("categories", "", "Categories to trigger (comma-separated). If none are specified, will use the .shipshape configuration file to decide which categories to run.")
	dind           = flag.Bool("inside_docker", false, "True if the CLI is run from inside a docker container")
	event          = flag.String("event", defaults.DefaultEvent, "The name of the event to use")
	jsonOutput     = flag.String("json_output", "", "When specified, log shipshape results to provided .json file")
	repo           = flag.String("repo", defaults.DefaultRepo, "The name of the docker repo to use")
	stayUp         = flag.Bool("stay_up", true, "True if we should keep the container running, false if we should stop and remove it.")
	tag            = flag.String("tag", "prod", "Tag to use for the analysis service image. If this is local, we will not attempt to pull the image.")
	useLocalKythe  = flag.Bool("local_kythe", false, "True if we should not pull down the kythe image. This is used for testing a new kythe image.")
	showCategories = flag.Bool("show_categories", false, "Show what categories are available instead of running analyses.")
	hotStart       = flag.Bool("hot_start", true, "Just start the service, but do nothing else.")
	keyFlags       = []string{"analyzer_images", "build", "categories", "inside_docker", "event", "json_output",
		"repo", "stay_up", "tag", "local_kythe", "show_categories"}
)

const (
	returnNoFindings = 0
	returnFindings   = 1
	returnError      = 2
)

func shipshapeUsage() {
	shipshapeArgs := make(map[string]bool)
	for _, flag := range keyFlags {
		shipshapeArgs[flag] = true
	}
	fmt.Println("USAGE: shipshape [flags] <directory>")
	fmt.Println("Shipshape flags: (for all flags, run shipshape -help)")
	flag.VisitAll(func(f *flag.Flag) {
		_, isShipshapeArg := shipshapeArgs[f.Name]
		if !isShipshapeArg {
			return
		}
		defValue := f.DefValue
		if defValue == "" {
			defValue = "\"\""
		}
		fmt.Printf("  -%s:\n\t %s (default: %s)\n", f.Name, f.Usage, defValue)
	})
}

func outputAsText(msg *rpcpb.ShipshapeResponse, directory string) error {
	// TODO(ciera): these results aren't sorted. They should be sorted by path and start line
	fileNotes := make(map[string][]*notepb.Note)
	for _, analysis := range msg.AnalyzeResponse {
		for _, failure := range analysis.Failure {
			fmt.Printf("WARNING: Analyzer %s failed to run: %s\n", *failure.Category, *failure.FailureMessage)
		}
		for _, note := range analysis.Note {
			path := ""
			if note.Location != nil {
				path = filepath.Join(directory, note.Location.GetPath())
			}
			fileNotes[path] = append(fileNotes[path], note)
		}
	}

	for path, notes := range fileNotes {
		if path != "" {
			fmt.Println(path)
		} else {
			fmt.Println("Global")
		}
		for _, note := range notes {
			loc := ""
			subCat := ""
			if note.Subcategory != nil {
				subCat = ":" + *note.Subcategory
			}
			if note.GetLocation().Range != nil && note.GetLocation().GetRange().StartLine != nil {
				if note.GetLocation().GetRange().StartColumn != nil {
					loc = fmt.Sprintf("Line %d, Col %d ", *note.Location.Range.StartLine, *note.Location.Range.StartColumn)
				} else {
					loc = fmt.Sprintf("Line %d ", *note.Location.Range.StartLine)
				}
			}

			fmt.Printf("%s[%s%s]\n", loc, *note.Category, subCat)
			fmt.Printf("\t%s\n", *note.Description)
		}
		fmt.Println()
	}
	return nil
}

func main() {
	flag.Parse()

	// Get the file/directory to analyze.
	// If we are just showing category list, default to the current directory
	file := "."
	if len(flag.Args()) >= 1 {
		file = flag.Arg(0)
	} else if !*showCategories {
		shipshapeUsage()
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
		File:                file,
		ThirdPartyAnalyzers: thirdPartyAnalyzers,
		Build:               *build,
		TriggerCats:         cats,
		Dind:                *dind,
		Event:               *event,
		Repo:                *repo,
		StayUp:              *stayUp,
		Tag:                 *tag,
		LocalKythe:          *useLocalKythe,
	}
	if *jsonOutput == "" {
		options.HandleResponse = outputAsText
	} else {
		// TODO(supertri): Does not work for showCategories
		var allResponses rpcpb.ShipshapeResponse
		options.HandleResponse = func(msg *rpcpb.ShipshapeResponse, _ string) error {
			allResponses.AnalyzeResponse = append(allResponses.AnalyzeResponse, msg.AnalyzeResponse...)
			return nil
		}
		options.ResponsesDone = func() error {
			// TODO(ciera): these results aren't sorted. They should be sorted by path and start line
			b, err := json.Marshal(allResponses)
			if err != nil {
				return err
			}
			return ioutil.WriteFile(*jsonOutput, b, 0644)
		}
	}
	invocation := cli.New(options)
	numResults := 0
	var err error = nil

	if *showCategories {
		err = invocation.ShowCategories()
	} else if *hotStart {
		err = invocation.StartService()
	} else {
		numResults, err = invocation.Run()
	}

	if err != nil {
		fmt.Printf("Error: %v", err.Error())
		os.Exit(returnError)
	}
	if numResults != 0 {
		os.Exit(returnFindings)
	}
	os.Exit(returnNoFindings)
}

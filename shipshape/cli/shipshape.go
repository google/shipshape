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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"shipshape/service"
	"shipshape/util/docker"
	"shipshape/util/rpc/client"
	glog "third_party/go-glog"

	"github.com/golang/protobuf/proto"

	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
)

var (
	tag     = flag.String("tag", "prod", "Tag to use for the analysis service image. If this is local, we will not attempt to pull the image.")
	streams = flag.Bool("streams", false, "True if we should run in streams mode, false if we should run as a service.")

	event          = flag.String("event", "manual", "The name of the event to use")
	categories     = flag.String("categories", "", "Categories to trigger (comma-separated). If none are specified, will use the .shipshape configuration file to decide which categories to run.")
	stayUp         = flag.Bool("stay_up", true, "True if we should keep the container running, false if we should stop and remove it.")
	repo           = flag.String("repo", "gcr.io/shipshape_releases", "The name of the docker repo to use")
	analyzerImages = flag.String("analyzer_images", "", "Full docker path to images of external analyzers to use (comma-separated)")
	jsonOutput     = flag.String("json_output", "", "When specified, log shipshape results to provided .json file")
	build          = flag.String("build", "", "The name of the build system to use to generate compilation units. If empty, will not run the compilation step. Options are maven and go.")
	useLocalKythe  = flag.Bool("local_kythe", false, "True if we should not pull down the kythe image. This is used for testing a new kythe image.")
)

const (
	workspace  = "/shipshape-workspace"
	logsDir    = "/shipshape-output"
	localLogs  = "/tmp"
	image      = "service"
	kytheImage = "kythe"
)

func logMessage(msg *rpcpb.ShipshapeResponse) error {
	if *jsonOutput == "" {
		fileNotes := make(map[string][]*notepb.Note)
		for _, analysis := range msg.AnalyzeResponse {
			for _, failure := range analysis.Failure {
				fmt.Printf("WARNING: Analyzer %s failed to run: %s\n", *failure.Category, *failure.FailureMessage)
			}
			for _, note := range analysis.Note {
				path := ""
				if note.Location != nil {
					path = note.Location.GetPath()
				}
				fileNotes[path] = append(fileNotes[path], note)
			}
		}

		// TODO(ciera): these results aren't sorted. They should be sorted by path and start line
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

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(*jsonOutput, b, 0644)
}

func main() {
	flag.Parse()

	// Get the directory to analyze.
	if len(flag.Args()) != 1 {
		fmt.Println("Usage: shipshape [OPTIONS] <directory>")
		return
	}

	dir := flag.Arg(0)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		fmt.Println("%s is not a valid directory", dir)
		return
	}

	absRoot, err := filepath.Abs(dir)
	if err != nil {
		fmt.Println("Could not get absolute path for %s: %v", dir, err)
		return
	}

	image := docker.FullImageName(*repo, image, *tag)
	glog.Infof("Starting shipshape using %s on %s", image, absRoot)

	// Create the request

	var trigger []string
	if *categories != "" {
		trigger = strings.Split(*categories, ",")
	} else {
		glog.Infof("No categories provided. Will be using categories specified by the config file for the event %s", *event)
	}

	req := &rpcpb.ShipshapeRequest{
		TriggeredCategory: trigger,
		ShipshapeContext: &ctxpb.ShipshapeContext{
			RepoRoot: proto.String(workspace),
		},
		Event: proto.String(*event),
		Stage: ctxpb.Stage_PRE_BUILD.Enum(),
	}

	var thirdPartyAnalyzers []string
	if *analyzerImages == "" {
		thirdPartyAnalyzers, err = service.GlobalConfig(absRoot)
		if err != nil {
			glog.Infof("Could not get global config; using only the default analyzers: %v", err)
		}
	} else {
		thirdPartyAnalyzers = strings.Split(*analyzerImages, ",")
	}

	// If we are not running in local mode, pull the latest copy
	// Notice this will use the local tag as a signal to not pull the
	// third-party analyzers either.
	if *tag != "local" {
		pull(image)
		pullAnalyzers(thirdPartyAnalyzers)
	}

	// Put in this defer before calling run. Even if run fails, it can
	// still create the container.
	if !*stayUp {
		// TODO(ciera): Rather than immediately sending a SIGKILL,
		// we should use the default 10 seconds and properly handle
		// SIGTERMs in the endpoint script.
		defer stop("shipping_container", 0)
		// Stop all the analyzers, even the ones that had trouble starting,
		// in case they did actually start
		for id, analyzerRepo := range thirdPartyAnalyzers {
			container, _ := getContainerAndAddress(analyzerRepo, id)
			defer stop(container, 0)
		}
	}

	containers, errs := startAnalyzers(absRoot, thirdPartyAnalyzers)
	for _, err := range errs {
		glog.Errorf("Could not start up third party analyzer: %v", err)
	}

	var c *client.Client

	// Run it on files
	if *streams {
		err = streamsAnalyze(image, absRoot, containers, req)
		if err != nil {
			glog.Errorf("Error making stream call: %v", err)
			return
		}
	} else {
		c, err = startShipshapeService(image, absRoot, containers)
		if err != nil {
			glog.Errorf("HTTP client did not become healthy: %v", err)
			return
		}
		err = serviceAnalyze(c, req)
		if err != nil {
			glog.Errorf("Error making service call: %v", err)
			return
		}
	}

	// If desired, generate compilation units with a kythe image
	if *build != "" {
		// TODO(ciera): Handle other build systems
		fullKytheImage := docker.FullImageName(*repo, kytheImage, *tag)
		if !*useLocalKythe {
			pull(fullKytheImage)
		}

		defer stop("kythe", 10*time.Second)
		glog.Infof("Retrieving compilation units with %s", *build)

		result := docker.RunKythe(fullKytheImage, "kythe", absRoot, *build)
		if result.Err != nil {
			// kythe spews output, so only capture it if something went wrong.
			printStreams(result)
			glog.Errorf("Error from run: %v", result.Err)
			return
		}
		glog.Infoln("CompilationUnits prepared")

		req.Stage = ctxpb.Stage_POST_BUILD.Enum()
		if !*streams {
			err = serviceAnalyze(c, req)
			if err != nil {
				glog.Errorf("Error making service call: %v", err)
				return
			}
		} else {
			err = streamsAnalyze(image, absRoot, containers, req)
			if err != nil {
				glog.Errorf("Error making stream call: %v", err)
				return
			}
		}
	}

	glog.Infoln("End of Results.")
}

func startShipshapeService(image, absRoot string, analyzers []string) (*client.Client, error) {
	// If this doesn't match the image, stop and restart the service.
	// Otherwise, use the existing one.
	if !docker.ImageMatches(image, "shipping_container") {
		glog.Infof("Restarting container with %s", image)
		stop("shipping_container", 0)
		result := docker.RunService(image, "shipping_container", absRoot, localLogs, analyzers)
		printStreams(result)
		if result.Err != nil {
			return nil, result.Err
		}
	}
	glog.Infof("Image %s running in service mode", image)
	c := client.NewHTTPClient("localhost:10007")
	return c, c.WaitUntilReady(10 * time.Second)
}

func serviceAnalyze(c *client.Client, req *rpcpb.ShipshapeRequest) error {
	glog.Infof("Calling to the shipshape service with %v", req)
	rd := c.Stream("/ShipshapeService/Run", req)
	defer rd.Close()
	for {
		var msg rpcpb.ShipshapeResponse
		if err := rd.NextResult(&msg); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("received an error from calling run: %v", err.Error())
		}

		if err := logMessage(&msg); err != nil {
			return fmt.Errorf("could not parse results: %v", err.Error())
		}
	}
	return nil
}

func streamsAnalyze(image, absRoot string, analyzerContainers []string, req *rpcpb.ShipshapeRequest) error {
	glog.Infof("Running image %s in stream mode", image)
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshalling %v: %v", req, err)
	}

	stop("shipping_container", 0)
	result := docker.RunStreams(image, "shipping_container", absRoot, localLogs, analyzerContainers, reqBytes)
	glog.Infoln(strings.TrimSpace(result.Stderr))

	if result.Err != nil {
		return fmt.Errorf("error from run: %v", result.Err)
	}
	var msg rpcpb.ShipshapeResponse
	if err := proto.Unmarshal([]byte(result.Stdout), &msg); err != nil {
		return fmt.Errorf("unexpected ShipshapeResponse %v", err)
	}
	return logMessage(&msg)
}

func pull(image string) {
	if !docker.OutOfDate(image) {
		return
	}
	glog.Infof("Pulling image %s", image)
	result := docker.Pull(image)
	printStreams(result)
	if result.Err != nil {
		glog.Errorf("Error from pull: %v", result.Err)
		return
	}
	glog.Infoln("Pulling complete")
}

func stop(container string, timeWait time.Duration) {
	glog.Infof("Stopping and removing %s", container)
	result := docker.Stop(container, timeWait, true)
	printStreams(result)
	if result.Err != nil {
		glog.Infof("Could not stop %s: %v", container, result.Err)
	} else {
		glog.Infoln("Removed.")
	}
}

func pullAnalyzers(images []string) {
	var wg sync.WaitGroup
	for _, analyzerRepo := range images {
		wg.Add(1)
		go func() {
			pull(analyzerRepo)
			wg.Done()
		}()
	}
	glog.Info("Pulling dockerized analyzers...")
	wg.Wait()
	glog.Info("Analyzers pulled")
}

func startAnalyzers(sourceDir string, images []string) (containers []string, errs []error) {
	var wg sync.WaitGroup
	for id, fullImage := range images {
		wg.Add(1)
		go func() {
			analyzerContainer, port := getContainerAndAddress(fullImage, id)
			result := docker.RunAnalyzer(fullImage, analyzerContainer, sourceDir, localLogs, port)
			if result.Err != nil {
				glog.Infof("Could not start %v at localhost:%d: %v", fullImage, port, result.Err.Error())
				errs = append(errs, result.Err)
			} else {
				glog.Infof("Analyzer %v started at localhost:%d", fullImage, port)
				containers = append(containers, analyzerContainer)
			}
			wg.Done()
		}()
	}
	glog.Info("Waiting for dockerized analyzers to start up...")
	wg.Wait()
	glog.Info("Analyzers up")
	return containers, errs
}

func printStreams(result docker.CommandResult) {
	out := strings.TrimSpace(result.Stdout)
	err := strings.TrimSpace(result.Stderr)
	if len(out) > 0 {
		glog.Infof("stdout:\n%s\n", strings.TrimSpace(result.Stdout))
	}
	if len(err) > 0 {
		glog.Infof("stderr:\n%s\n", strings.TrimSpace(result.Stderr))
	}
}

func getContainerAndAddress(fullImage string, id int) (analyzerContainer string, port int) {
	end := strings.LastIndex(fullImage, ":")
	if end == -1 {
		end = len(fullImage) - 1
	}
	image := fullImage[strings.LastIndex(fullImage, "/")+1 : end]
	port = 10010 + id
	analyzerContainer = fmt.Sprintf("%s_%d", image, id)
	return analyzerContainer, port
}

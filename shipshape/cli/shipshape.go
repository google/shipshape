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
	analyzerImages = flag.String("analyzer_images", "", "Full docker path to images of external analyzers to use (comma-separated)")
	build          = flag.String("build", "", "The name of the build system to use to generate compilation units. If empty, will not run the compilation step. Options are maven and go.")
	categories     = flag.String("categories", "", "Categories to trigger (comma-separated). If none are specified, will use the .shipshape configuration file to decide which categories to run.")
	dind           = flag.Bool("inside_docker", false, "True if the CLI is run from inside a docker container")
	event          = flag.String("event", "manual", "The name of the event to use")
	jsonOutput     = flag.String("json_output", "", "When specified, log shipshape results to provided .json file")
	repo           = flag.String("repo", "gcr.io/shipshape_releases", "The name of the docker repo to use")
	stayUp         = flag.Bool("stay_up", true, "True if we should keep the container running, false if we should stop and remove it.")
	streams        = flag.Bool("streams", false, "True if we should run in streams mode, false if we should run as a service.")
	tag            = flag.String("tag", "prod", "Tag to use for the analysis service image. If this is local, we will not attempt to pull the image.")
	useLocalKythe  = flag.Bool("local_kythe", false, "True if we should not pull down the kythe image. This is used for testing a new kythe image.")
)

const (
	workspace  = "/shipshape-workspace"
	logsDir    = "/shipshape-output"
	localLogs  = "/tmp"
	image      = "service"
	kytheImage = "kythe"
)

func logMessage(msg *rpcpb.ShipshapeResponse, directory string) error {
	if *jsonOutput == "" {
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
	glog.Infof("Starting shipshape...")
	flag.Parse()

	// Get the directory to analyze.
	if len(flag.Args()) != 1 {
		fmt.Println("Usage: shipshape [OPTIONS] <directory>")
		return
	}

	file := flag.Arg(0)
	fs, err := os.Stat(file)
	if err != nil {
		fmt.Printf("%s is not a valid file or directory\n", file)
		return
	}

	origDir := file
	if !fs.IsDir() {
		origDir = filepath.Dir(file)
	}

	absRoot, err := filepath.Abs(origDir)
	if err != nil {
		fmt.Printf("Could not get absolute path for %s: %v\n", origDir, err)
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

	containers, errs := startAnalyzers(absRoot, thirdPartyAnalyzers, *dind)
	for _, err := range errs {
		glog.Errorf("Could not start up third party analyzer: %v", err)
	}

	var c *client.Client
	var req *rpcpb.ShipshapeRequest

	// Run it on files
	if *streams {
		var files []string
		if !fs.IsDir() {
			files = []string{file}
		}
		req = createRequest(trigger, files, *event, workspace, ctxpb.Stage_PRE_BUILD.Enum())
		err = streamsAnalyze(image, absRoot, origDir, containers, req, *dind)
		if err != nil {
			glog.Errorf("Error making stream call: %v", err)
			return
		}
	} else {
		relativeRoot := ""
		c, relativeRoot, err = startShipshapeService(image, absRoot, containers, *dind)
		if err != nil {
			glog.Errorf("HTTP client did not become healthy: %v", err)
			return
		}
		var files []string
		if !fs.IsDir() {
			files = []string{filepath.Base(file)}
		}
		req = createRequest(trigger, files, *event, filepath.Join(workspace, relativeRoot), ctxpb.Stage_PRE_BUILD.Enum())
		err = serviceAnalyze(c, req, origDir)
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

		result := docker.RunKythe(fullKytheImage, "kythe", absRoot, *build, *dind)
		if result.Err != nil {
			// kythe spews output, so only capture it if something went wrong.
			printStreams(result)
			glog.Errorf("Error from run: %v", result.Err)
			return
		}
		glog.Infoln("CompilationUnits prepared")

		req.Stage = ctxpb.Stage_POST_BUILD.Enum()
		if !*streams {
			err = serviceAnalyze(c, req, origDir)
			if err != nil {
				glog.Errorf("Error making service call: %v", err)
				return
			}
		} else {
			err = streamsAnalyze(image, absRoot, origDir, containers, req, *dind)
			if err != nil {
				glog.Errorf("Error making stream call: %v", err)
				return
			}
		}
	}

	glog.Infoln("End of Results.")
}

// startShipshapeService ensures that there is a service started with the given image and
// attached analyzers that can analyze the directory at absRoot (an absolute path). If a
// service is not started up that can do this, it will shut down the existing one and start
// a new one.
// The methods returns the (ready) client, the relative path from the docker container's mapped
// volume to the absRoot that we are analyzing, and any errors from attempting to run the service.
// TODO(ciera): This *should* check the analyzers that are connected, but does not yet
// do so.
func startShipshapeService(image, absRoot string, analyzers []string, dind bool) (*client.Client, string, error) {
	glog.Infof("Starting shipshape...")
	container := "shipping_container"
	// subPath is the relatve path from the mapped volume on shipping container
	// to the directory we are analyzing (absRoot)
	isMapped, subPath := docker.MappedVolume(absRoot, container)
	// Stop and restart the container if:
	// 1: The container is not using the latest image OR
	// 2: The container is not mapped to the right directory OR
	// 3: The container is not linked to the right analyzer containers
	// Otherwise, use the existing container
	if !docker.ImageMatches(image, container) || !isMapped || !docker.ContainsLinks(container, analyzers) {
		glog.Infof("Restarting container with %s", image)
		stop(container, 0)
		result := docker.RunService(image, container, absRoot, localLogs, analyzers, dind)
		subPath = ""
		printStreams(result)
		if result.Err != nil {
			return nil, "", result.Err
		}
	}
	glog.Infof("Image %s running in service mode", image)
	c := client.NewHTTPClient("localhost:10007")
	return c, subPath, c.WaitUntilReady(10 * time.Second)
}

func serviceAnalyze(c *client.Client, req *rpcpb.ShipshapeRequest, originalDir string) error {
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

		if err := logMessage(&msg, originalDir); err != nil {
			return fmt.Errorf("could not parse results: %v", err.Error())
		}
	}
	return nil
}

func streamsAnalyze(image, absRoot, originalDir string, analyzerContainers []string, req *rpcpb.ShipshapeRequest, dind bool) error {
	glog.Infof("Running image %s in stream mode", image)
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshalling %v: %v", req, err)
	}

	stop("shipping_container", 0)
	result := docker.RunStreams(image, "shipping_container", absRoot, localLogs, analyzerContainers, reqBytes, dind)
	glog.Infoln(strings.TrimSpace(result.Stderr))

	if result.Err != nil {
		return fmt.Errorf("error from run: %v", result.Err)
	}
	var msg rpcpb.ShipshapeResponse
	if err := proto.Unmarshal([]byte(result.Stdout), &msg); err != nil {
		return fmt.Errorf("unexpected ShipshapeResponse %v", err)
	}
	return logMessage(&msg, originalDir)
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

func startAnalyzers(sourceDir string, images []string, dind bool) (containers []string, errs []error) {
	var wg sync.WaitGroup
	for id, fullImage := range images {
		wg.Add(1)
		go func() {
			analyzerContainer, port := getContainerAndAddress(fullImage, id)
			if docker.ImageMatches(fullImage, analyzerContainer) {
				glog.Infof("Reusing analyzer %v started at localhost:%d", fullImage, port)
			} else {
				glog.Infof("Found no analyzer container (%v) to reuse for %v", analyzerContainer, fullImage)
				// Analyzer is either running with the wrong image version, or not running
				// Stopping in case it's the first case
				result := docker.Stop(analyzerContainer, 0, true)
				if result.Err != nil {
					glog.Infof("Failed to stop %v (may not be running)", analyzerContainer)
				}
				result = docker.RunAnalyzer(fullImage, analyzerContainer, sourceDir, localLogs, port, dind)
				if result.Err != nil {
					glog.Infof("Could not start %v at localhost:%d: %v, stderr: %v", fullImage, port, result.Err.Error(), result.Stderr)
					errs = append(errs, result.Err)
				} else {
					glog.Infof("Analyzer %v started at localhost:%d", fullImage, port)
					containers = append(containers, analyzerContainer)
				}
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
		glog.Errorf("stderr:\n%s\n", strings.TrimSpace(result.Stderr))
	}
}

func getContainerAndAddress(fullImage string, id int) (analyzerContainer string, port int) {
	// A docker image URI (location:port/path:tag) can have a host part
	// with a port number and a path part with a tag.  Both tag and port
	// are separated by colon, so we need to find out if the last colon is
	// the one that separates the tag from the path, or the port in the
	// location.
	end := strings.LastIndex(fullImage, ":")
	slash := strings.LastIndex(fullImage, "/")
	if end == -1 || end < slash {
		// No colon, or last colon is part of the location.
		end = len(fullImage)
	}
	image := fullImage[slash+1 : end]
	port = 10010 + id
	analyzerContainer = fmt.Sprintf("%s_%d", image, id)
	return analyzerContainer, port
}

func createRequest(triggerCats, files []string, event, repoRoot string, stage *ctxpb.Stage) *rpcpb.ShipshapeRequest {
	return &rpcpb.ShipshapeRequest{
		TriggeredCategory: triggerCats,
		ShipshapeContext: &ctxpb.ShipshapeContext{
			RepoRoot: proto.String(repoRoot),
			FilePath: files,
		},
		Event: proto.String(event),
		Stage: stage,
	}
}

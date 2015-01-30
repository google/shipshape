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
	"time"

	"shipshape/util/docker"
	glog "third_party/go-glog"
	"third_party/kythe/go/rpc/client"

	"github.com/golang/protobuf/proto"

	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
)

var (
	tag     = flag.String("tag", "prod", "Tag to use for the analysis service image")
	local   = flag.Bool("try_local", false, "True if we should use the local copy of this image, and pull only if it doesn't exist. False will always pull.")
	streams = flag.Bool("streams", false, "True if we should run in streams mode, false if we should run as a service.")

	event      = flag.String("event", "manual", "The name of the event to use")
	categories = flag.String("categories", "", "Categories to trigger (comma-separated). If none are specified, will use the .shipshape configuration file to decide which categories to run.")
	stayUp     = flag.Bool("stay_up", false, "True if we should keep the container running for debugging purposes, false if we should stop and remove it.")
	repo       = flag.String("repo", "gcr.io/_b_shipshape_registry", "The name of the docker repo to use")
	kytheRepo  = flag.String("kytheRepo", "gcr.io/kythe_repo", "The name of the docker repo to use")
	// TODO(ciera): use the analyzer images
	//analyzerImages  = flag.String("analyzer_images", "", "Full docker path to images of external analyzers to use (comma-separated)")
	jsonOutput = flag.String("json_output", "", "When specified, log shipshape results to provided .json file")
	build      = flag.String("build", "", "The name of the build system to use to generate compilation units. If empty, will not run the compilation step. Options are maven and go.")
)

const (
	workspace = "/shipshape-workspace"
	logsDir   = "/shipshape-output"
	localLogs = "/tmp"
	image     = "service"
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
		glog.Fatal("Usage: shipshape [OPTIONS] <directory>")
	}

	dir := flag.Arg(0)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		glog.Fatalf("%s is not a valid directory", dir)
	}

	absRoot, err := filepath.Abs(dir)
	if err != nil {
		glog.Fatalf("Could not get absolute path for %s: %v", dir, err)
	}

	res := docker.Authenticate()
	if res.Err != nil {
		glog.Infoln(strings.TrimSpace(res.Stderr))
		glog.Fatalf("Could not authenticate: %v", res.Err)
	}
	glog.Infoln(strings.TrimSpace(res.Stdout))

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

	// If necessary, pull it
	// If local is true it doesn't meant that docker won't pull it, it will just
	// look locally first.
	if !*local {
		pull(image)
	}

	// Put in this defer before calling run. Even if run fails, it can
	// still create the container.
	if !*stayUp {
		defer stop("shipping_container")
	}

	var c *client.Client

	// Run it on files
	if *streams {
		err = streamsAnalyze(absRoot, req)
		if err != nil {
			glog.Errorf("Error making stream call: %v", err)
			return
		}
	} else {
		c, err = startShipshapeService(absRoot)
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
		// TODO(ciera): handle campfire as an option
		kytheImage := docker.FullImageName(*kytheRepo, "kythe", "latest")
		if !*local {
			pull(kytheImage)
		}

		defer stop("kythe")
		glog.Infof("Retrieving compilation units with %s", *build)
		volumeMap := map[string]string{
			filepath.Join(absRoot, "compilations"): "/compilations",
			absRoot: "/repo",
		}
		home := os.Getenv("HOME")
		if len(home) > 0 {
			volumeMap[filepath.Join(home, ".m2")] = "/root/.m2"
		} else {
			glog.Infof("$HOME is not set. Not using .m2 mapping")
		}

		// TODO(ciera): Can I pass in more than one extractor?
		// TODO(ciera): Can we exclude files in the .shipshape ignore path?
		// TODO(ciera): Can we use the same command for campfire extraction?
		result := docker.RunAttached(kytheImage, "kythe", nil, volumeMap, nil, nil, nil, []string{"--extract", *build})
		if result.Err != nil {
			// kythe spews output, so only capture it if something went wrong.
			glog.Infoln(strings.TrimSpace(result.Stdout))
			glog.Infoln(strings.TrimSpace(result.Stderr))
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
			err = streamsAnalyze(absRoot, req)
			if err != nil {
				glog.Errorf("Error making stream call: %v", err)
				return
			}
		}
	}

	glog.Infoln("End of Results.")
}

func startShipshapeService(absRoot string) (*client.Client, error) {
	volumeMap := map[string]string{absRoot: workspace, localLogs: logsDir}
	glog.Infof("Running image %s in service mode", image)
	environment := map[string]string{"START_SERVICE": "true"}
	result := docker.Run(image, "shipping_container", map[int]int{10007: 10007}, volumeMap, nil, environment, nil)
	glog.Infoln(strings.TrimSpace(result.Stdout))
	glog.Infoln(strings.TrimSpace(result.Stderr))
	if result.Err != nil {
		return nil, result.Err
	}
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

func streamsAnalyze(absRoot string, req *rpcpb.ShipshapeRequest) error {
	volumeMap := map[string]string{absRoot: workspace, localLogs: logsDir}
	glog.Infof("Running image %s in stream mode", image)
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshalling %v: %v", req, err)
	}

	result := docker.RunAttached(image, "shipping_container", map[int]int{10007: 10007}, volumeMap, nil, nil, reqBytes, nil)
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
	glog.Infof("Pulling image %s", image)
	result := docker.Pull(image)
	glog.Infoln(strings.TrimSpace(result.Stdout))
	glog.Infoln(strings.TrimSpace(result.Stderr))
	if result.Err != nil {
		glog.Errorf("Error from pull: %v", result.Err)
		return
	}
	glog.Infoln("Pulling complete")
}

func stop(container string) {
	glog.Infof("Stopping and removing %s", container)
	result := docker.Stop(container, true)
	glog.Infoln(strings.TrimSpace(result.Stdout))
	glog.Infoln(strings.TrimSpace(result.Stderr))
	if result.Err != nil {
		glog.Infof("Could not stop %s: %v", container, result.Err)
	} else {
		glog.Infoln("Removed.")
	}
}

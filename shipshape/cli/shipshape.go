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
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"third_party/kythe/go/rpc/client"
	"shipshape/util/docker"

	"code.google.com/p/goprotobuf/proto"

	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
	spb "shipshape/proto/source_context_proto"
)

var (
	tag     = flag.String("tag", "prod", "Tag to use for the analysis service image")
	local   = flag.Bool("try_local", false, "True if we should use the local copy of this image, and pull only if it doesn't exist. False will always pull.")
	streams = flag.Bool("streams", false, "True if we should run in streams mode, false if we should run as a service.")

	event      = flag.String("event", "TestClient", "The name of the event to use")
	categories = flag.String("categories", "", "Categories to trigger (comma-separated). If none are specified, will use the .shipshape configuration file to decide which categories to run.")
	stayUp     = flag.Bool("stay_up", false, "True if we should keep the container running for debugging purposes, false if we should stop and remove it.")
	repo       = flag.String("repo", "container.cloud.google.com/_b_shipshape_registry", "The name of the docker repo to use")
	// TODO(ciera): use the analyzer images
	//analyzerImages  = flag.String("analyzer_images", "", "Full docker path to images of external analyzers to use (comma-separated)")
	jsonOutput = flag.String("json_output", "", "When specified, log shipshape results to provided .json file")
)

const (
	workspace = "/shipshape-workspace"
	logsDir   = "/shipshape-output"
	localLogs = "/tmp"
	image     = "service"
)

func logMessage(msg *rpcpb.ShipshapeResponse) error {
	if *jsonOutput == "" {
		log.Printf("result:\n\n%v\n", proto.MarshalTextString(msg))
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

	// 0. Get the directory to analyze.
	if len(flag.Args()) != 1 {
		log.Fatal("Usage: shipshape [OPTIONS] <directory>")
	}

	dir := flag.Arg(0)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		log.Fatalf("%s is not a valid directory", dir)
	}

	absRoot, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("Could not get absolute path for %s: %v", dir, err)
	}

	image := docker.FullImageName(*repo, image, *tag)
	log.Printf("Starting shipshape using %s on %s", image, absRoot)

	// 1. Create the request
	// TODO(ciera): What should we do for a local run?
	// Consider using a LocalContext in SourceContext, or putting a oneof
	// in the Shipshape location.
	sourceContext := &spb.SourceContext{}

	var trigger []string
	if *categories != "" {
		trigger = strings.Split(*categories, ",")
	} else {
		log.Printf("No categories found. Will be using categories specified by the config file for the event %s", *event)
	}

	req := &rpcpb.ShipshapeRequest{
		TriggeredCategory: trigger,
		ShipshapeContext: &ctxpb.ShipshapeContext{
			SourceContext: sourceContext,
			RepoRoot:      proto.String(workspace),
		},
		Event: proto.String(*event),
	}
	log.Printf("Using request:\n%v\n", req)

	// 2. If necessary, pull it
	// If local is true it doesn't meant that docker won't pull it, it will just
	// look locally first.
	if !*local {
		log.Printf("Pulling image %s", image)
		result := docker.Pull(image)
		fmt.Println(strings.TrimSpace(result.Stdout))
		if result.Err != nil {
			log.Println(strings.TrimSpace(result.Stderr))
			log.Panicf("Error from pull: %v", result.Err)
		}
		log.Println("Pulling complete")
	}

	volumeMap := map[string]string{absRoot: workspace, localLogs: logsDir}

	// Put in this defer before calling run. Even if run fails, it can
	// still create the container.
	if !*stayUp {
		defer func() {
			log.Println("Stopping and removing shipping_container")
			result := docker.Stop("shipping_container", true)
			fmt.Println(strings.TrimSpace(result.Stdout))
			if result.Err != nil {
				log.Println(strings.TrimSpace(result.Stderr))
				log.Printf("Could not stop shipping_container: %v", result.Err)
			} else {
				log.Println("Removed.")
			}
		}()
	}

	// 3. Run it!
	if *streams {
		log.Printf("Running image %s in stream mode", image)
		reqBytes, err := proto.Marshal(req)
		if err != nil {
			log.Panicf("Error marshalling %v: %v", req, err)
		}

		result := docker.RunAttached(image, "shipping_container", map[int]int{10007: 10007}, volumeMap, nil, nil, reqBytes)
		log.Println(strings.TrimSpace(result.Stderr))

		if result.Err != nil {
			log.Panicf("Error from run: %v", result.Err)
		}
		var msg rpcpb.ShipshapeResponse
		if err := proto.Unmarshal([]byte(result.Stdout), &msg); err != nil {
			log.Panicf("Unexpected ShipshapeResponse %v", err)
		}
		err = logMessage(&msg)
		if err != nil {
			log.Panicf("Error processing results: %v", err)
		}
	} else {
		log.Printf("Running image %s in service mode", image)
		environment := map[string]string{"START_SERVICE": "true"}
		result := docker.Run(image, "shipping_container", map[int]int{10007: 10007}, volumeMap, nil, environment)
		fmt.Println(strings.TrimSpace(result.Stdout))
		log.Println(strings.TrimSpace(result.Stderr))
		if result.Err != nil {
			log.Panicf("Error from run: %v", result.Err)
		}
		log.Println("Image running")

		log.Println("About to call out to the shipshape service")
		c := client.NewHTTPClient("localhost:10007")
		if err := c.WaitUntilReady(10 * time.Second); err != nil {
			log.Panicf("HTTP client did not become healthy: %v", err)
		}
		rd := c.Stream("/ShipshapeService/Run", req)
		defer rd.Close()
		for {
			var msg rpcpb.ShipshapeResponse
			if err := rd.NextResult(&msg); err == io.EOF {
				break
			} else if err != nil {
				log.Panicf("Error from proto call: %v", err)
			}

			if err := logMessage(&msg); err != nil {
				log.Panicf("Error processing results: %v", err)
			}
		}
	}

	log.Printf("End of Results.")
}

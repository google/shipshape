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

// Binary test_analyzer_client is a testing client that initiates a call
// to a shipshape analyzer. It assumes that a shipshape analyzer service
// has already been started at the specified ports
package main

import (
	"flag"
	"io"
	"log"
	"strings"

	"shipshape/util/file"
	"third_party/kythe/go/rpc/client"

	"code.google.com/p/goprotobuf/proto"

	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
	spb "shipshape/proto/source_context_proto"
)

var (
	servicePort = flag.String("service_port", "localhost:10007", "Service port")
	filePaths   = flag.String("files", "", "List of files (comma-separated)")
	projectName = flag.String("project_name", "quixotic-treat-519", "Project name of a Cloud project")
	hash        = flag.String("hash", "master", "Hash in repo for Cloud project")
	categories  = flag.String("categories", "", "Categories to trigger (comma-separated)")
	repoKind    = flag.String("repo_kind", "LOCAL", "The repo kind (CLOUD or LOCAL)")
	repoBase    = flag.String("repo_base", "/tmp", "The root of the repo to use, if LOCAL or the base directory to copy repo into if not LOCAL")
	volumeName  = flag.String("volume_name", "/shipshape-workspace", "The name of the shipping_container volume")
	event       = flag.String("event", "TestClient", "The name of the event to use")
	stage       = flag.String("stage", "PRE_BUILD", "The stage to test, either PRE_BUILD or POST_BUILD")
)

const (
	cloudRepo = "CLOUD"
	local     = "LOCAL"
	// TODO(supertri): Add GERRIT-ON-BORG when we implement for copying down
)

// TODO(supertri): utility methods to create request protos shared with test_analyzer_client?
func main() {
	flag.Parse()

	var root = ""
	var sourceContext *spb.SourceContext
	var err error

	switch *repoKind {
	case cloudRepo:
		sourceContext, root, err = file.SetupCloudRepo(*projectName, *hash, *repoBase, *volumeName)
		if err != nil {
			log.Fatalf("Failed to setup Cloud repo: %v", err)
		}
	case local:
		sourceContext = &spb.SourceContext{
			CloudRepo: &spb.CloudRepoSourceContext{
				RepoId: &spb.RepoId{
					ProjectRepoId: &spb.ProjectRepoId{
						ProjectId: projectName,
					},
				},
				RevisionId: hash,
			},
		}
		if *repoBase == "/tmp" {
			log.Fatal("Must specify the repo_base for local runs")
		}
		root = *repoBase
	default:
		log.Fatalf("Invalid repo kind %q", *repoKind)
	}

	var trigger = []string(nil)

	if len(*categories) > 0 {
		trigger = strings.Split(*categories, ",")
	}

	c := client.NewHTTPClient(*servicePort)
	var paths []string
	if *filePaths != "" {
		paths = strings.Split(*filePaths, ",")
	}
	var stageEnum ctxpb.Stage
	switch *stage {
	case "POST_BUILD":
		stageEnum = ctxpb.Stage_POST_BUILD
	case "PRE_BUILD":
		stageEnum = ctxpb.Stage_PRE_BUILD
	default:
		log.Fatalf("Invalid stage %q", *stage)
	}

	req := &rpcpb.ShipshapeRequest{
		TriggeredCategory: trigger,
		ShipshapeContext: &ctxpb.ShipshapeContext{
			FilePath:      paths,
			SourceContext: sourceContext,
			RepoRoot:      proto.String(root),
		},
		Event: proto.String(*event),
		Stage: stageEnum.Enum(),
	}
	log.Println("About to call out to the shipshape service")

	rd := c.Stream("/ShipshapeService/Run", req)
	defer rd.Close()
	for {
		var msg rpcpb.ShipshapeResponse
		if err := rd.NextResult(&msg); err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("Error from call: %v", err)
		}
		log.Printf("result:\n\n%v\n", proto.MarshalTextString(&msg))
	}

	log.Printf("Done.")
}

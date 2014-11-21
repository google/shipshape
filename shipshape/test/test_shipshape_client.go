// Binary test_analyzer_client is a testing client that initiates a call
// to a shipshape analyzer. It assumes that a shipshape analyzer service
// has already been started at the specified ports
package main

import (
	"flag"
	"io"
	"log"
	"strings"

	"third_party/kythe/go/rpc/client"
	"shipshape/util/file"

	"code.google.com/p/goprotobuf/proto"

	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
	spb "shipshape/proto/source_context_proto"
)

var (
	servicePort = flag.String("service_port", "localhost:10007", "Service port")
	filePaths   = flag.String("files", "README.md,sample.js,pypackage/foo.py,android-sample/MainActivity/src/com/example/android/tictactoe/MainActivity.java", "List of files (comma-separated)")
	projectName = flag.String("project_name", "quixotic-treat-519", "Project name of a Cloud project")
	hash        = flag.String("hash", "master", "Hash in repo for Cloud project")
	categories  = flag.String("categories", "", "Categories to trigger (comma-separated)")
	repoKind    = flag.String("repo_kind", "LOCAL", "The repo kind (CLOUD or LOCAL)")
	repoBase    = flag.String("repo_base", "/tmp", "The root of the repo to use, if LOCAL or the base directory to copy repo into if not LOCAL")
	volumeName  = flag.String("volume_name", "/shipshape-workspace", "The name of the shipping_container volume")
	event       = flag.String("event", "TestClient", "The name of the event to use")
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

	if *repoKind == cloudRepo {
		sourceContext, root, err = file.SetupCloudRepo(*projectName, *hash, *repoBase, *volumeName)
		if err != nil {
			log.Fatalf("Failed to setup Cloud repo: %v", err)
		}
	} else if *repoKind == local {
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
	} else {
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
	req := &rpcpb.ShipshapeRequest{
		TriggeredCategory: trigger,
		ShipshapeContext: &ctxpb.ShipshapeContext{
			FilePath:      paths,
			SourceContext: sourceContext,
			RepoRoot:      proto.String(root),
		},
		Event: proto.String(*event),
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

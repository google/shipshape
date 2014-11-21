// Binary test_analyzer_client is a testing client that initiates a call
// to a shipshape analyzer. It assumes that a shipshape analyzer service
// has already been started at the specified ports
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"third_party/kythe/go/platform/indexinfo"
	"third_party/kythe/go/rpc/client"
	"shipshape/util/file"

	"code.google.com/p/goprotobuf/proto"

	apb "third_party/kythe/proto/analysis_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
	spb "shipshape/proto/source_context_proto"
)

var (
	servicePort = flag.String("service_port", "localhost:10005", "Service port")
	filePaths   = flag.String("files", "README.md,sample.js,pypackage/foo.py,android-sample/MainActivity/src/com/example/android/tictactoe/MainActivity.java", "List of files (comma-separated)")
	projectName = flag.String("project_name", "quixotic-treat-519", "Project name of a Cloud project")
	hash        = flag.String("hash", "master", "Hash in repo for Cloud project")
	categories  = flag.String("categories", "", "Categories to trigger (comma-separated)")
	repoKind    = flag.String("repo_kind", "LOCAL", "The repo kind (CLOUD or LOCAL)")
	repoBase    = flag.String("repo_base", "/tmp", "The root of the repo to use, if LOCAL or the base directory to copy repo into if not LOCAL")
	volumeName  = flag.String("volume_name", "/shipshape-workspace", "The name of the shipping_container volume")
)

const (
	cloudRepo     = "CLOUD"
	local         = "LOCAL"
	analyzeMethod = "/AnalyzerService/Analyze"
	// TODO(supertri): Add GERRIT-ON-BORG when we implement for copying down
)

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

	c := client.NewHTTPClient(*servicePort)

	var paths []string
	if *filePaths != "" {
		paths = strings.Split(*filePaths, ",")
	}

	req := &rpcpb.AnalyzeRequest{
		Category: strings.Split(*categories, ","),
		ShipshapeContext: &ctxpb.ShipshapeContext{
			FilePath:      paths,
			SourceContext: sourceContext,
			RepoRoot:      proto.String(root),
		},
	}

	log.Printf("About to call out to the analyzer service without comp units. Request %v", req)
	readResponse(c, req)

	// Want some .kindex files?
	// Run
	// mkdir <repoRoot>/compilations
	// docker run --rm -v "<repoRoot>/compilations:/repo" -v "<repoRoot>":"/compilations" \
	// google/kythe-extractor-campfire Javac,GoCompile kythe/data/vnames.json
	// Or, for a simple maven project:
	// ./campfire-out/bin/kythe/java/com/google/devtools/kythe/extractors/maven/standalone <dir>
	compUnits, err := compUnitsFrom(root)
	if err != nil {
		log.Fatalf("Could not read kindex files: %v", err)
	}
	for path, unit := range compUnits {
		req.ShipshapeContext.CompilationDetails = &ctxpb.CompilationDetails{
			CompilationUnit:            unit,
			CompilationDescriptionPath: proto.String(path),
		}
		log.Printf("About to call out to the analyzer service with comp unit. Request %v", req)
		readResponse(c, req)
	}
	log.Print("Done.")
}

func readResponse(c *client.Client, req *rpcpb.AnalyzeRequest) {
	rd := c.Stream(analyzeMethod, req)
	defer rd.Close()
	for {
		var msg rpcpb.AnalyzeResponse
		if err := rd.NextResult(&msg); err == io.EOF {
			return
		} else if err != nil {
			log.Fatalf("Call to analyze failed with error: %v. Request: %v", err, req)
			return
		}
		fmt.Printf("result:\n\n%v\n", proto.MarshalTextString(&msg))
	}
}

// func compUnitsFrom pulls out all the compilation units that exist recursively under dir.
// It presumes that all such compilation units are contained in a .kindex file. It then uses
// indexinfo to extract the compilation unit. It returns a mapping of paths to compilation units.
func compUnitsFrom(dir string) (map[string]*apb.CompilationUnit, error) {
	// TODO(ciera): kindex files are (currently) divided by language
	// and located within a single directory. Figure out how to handle
	// this better in the test.
	var units = make(map[string]*apb.CompilationUnit)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return units, err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".kindex") {
			continue
		}
		path := filepath.Join(dir, file.Name())
		info, err := indexinfo.Open(path)
		if err != nil {
			return units, err
		}
		units[path] = info.Compilation
	}

	return units, nil
}

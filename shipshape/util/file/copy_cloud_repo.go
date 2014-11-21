// Binary copy_cloud_repo copies down a cloud repo to a location
// on disk. It returns the location of the root of the copied repo on stdout.
package main

import (
	"flag"
	"fmt"
	"log"

	"shipshape/util/file"
)

var (
	projectName  = flag.String("project_name", "quixotic-treat-519", "Project name of a Cloud project")
	hash         = flag.String("hash", "master", "Hash in repo for Cloud project")
	copyLocation = flag.String("copy_location", "/tmp", "The location to copy Cloud repo to")
)

func main() {
	flag.Parse()
	if *projectName == "" || *hash == "" {
		log.Fatal("Need to specify projectName and hash flags for a Cloud repo")
	}
	location, err := file.CopyCloudRepo(*projectName, *hash, *copyLocation)
	if err != nil {
		log.Fatalf("Could not copy down Cloud repo: %v", err)
	}
	fmt.Printf("COPY_LOCATION:%s\n", location)
}

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

// Binary copy_cloud_repo copies down a cloud repo to a location
// on disk. It returns the location of the root of the copied repo on stdout.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/google/shipshape/shipshape/util/file"
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

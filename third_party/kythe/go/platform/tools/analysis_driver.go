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

// Binary analysis_driver reads --index_info files from a glob (or from os.Stdin
// as one filepath per line) and sends each CompilationUnit to the specified
// --analyzer address. Before sending any requests, analysis_driver will start a
// local FileData service on --fds_port. The driver will populate each analysis
// request with the local FileData service's address which will serve the files
// from each indexinfo file.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"third_party/kythe/go/platform/delimited"
	"third_party/kythe/go/platform/indexinfo"
	"third_party/kythe/go/rpc/client"
	"third_party/kythe/go/rpc/server"
	"third_party/kythe/go/storage/files"

	apb "third_party/kythe/proto/analysis_proto"
	spb "third_party/kythe/proto/storage_proto"
)

var (
	analyzer = flag.String("analyzer", "", "Address of analyzer for which to send requests (required)")
	fdsPort  = flag.Int("fds_port", 0, "Port to serve file data (0 to choose dynamically)")
)

func main() {
	flag.Parse()
	log.SetPrefix("analysis_driver: ")

	if *analyzer == "" {
		log.Fatal("Must provide --analyzer address")
	} else if *fdsPort < 0 {
		log.Fatalf("Invalid --fds_port %d", *fdsPort)
	}

	idxFiles := make(chan string)
	go func() {
		defer close(idxFiles)
		if len(flag.Args()) == 0 {
			s := bufio.NewScanner(os.Stdin)
			for s.Scan() {
				idxFiles <- s.Text()
			}
			if s.Err() != nil {
				log.Fatal(s.Err())
			}
		} else {
			for _, arg := range flag.Args() {
				matches, err := filepath.Glob(arg)
				if err != nil {
					log.Fatalf("Error globing indexinfo files: %v", err)
				}
				for _, file := range matches {
					idxFiles <- file
				}
			}
		}
	}()

	fileStore := files.InMemory()
	s := server.Service{Name: "FileData"}
	if err := s.Register(&files.FileDataService{fileStore}); err != nil {
		log.Fatalf("Could not register FileData service: %v", err)
	}
	http.Handle("/", server.Endpoint{&s})

	host, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	// Explicitly get a Listener to determine chosen port
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", *fdsPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", *fdsPort, err)
	}
	go func() {
		if err := http.Serve(l, nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	c := client.NewHTTPClient(*analyzer)

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	fdsAddress := fmt.Sprintf("%s:%s", host, port)

	var errors int
	for idxFile := range idxFiles {
		idx, err := indexinfo.Open(idxFile)
		if err != nil {
			log.Fatalf("Error opening indexinfo: %v", err)
		}

		if err := fileStore.AddData(idx.Files...); err != nil {
			log.Fatalf("Error adding files to FileDataService: %v", err)
		}

		req := &apb.AnalysisRequest{
			Compilation:     idx.Compilation,
			FileDataService: &fdsAddress,
		}

		wr := delimited.NewWriter(os.Stdout)

		log.Printf("Analyzing %q", idxFile)
		rd := c.Stream("/CompilationAnalyzer/Analyze", req)
		for {
			var entry spb.Entry
			if err := rd.NextResult(&entry); err == io.EOF {
				break
			} else if err != nil {
				log.Printf("Analysis failure: %v", err)
				errors++
				break
			}
			if err := wr.PutProto(&entry); err != nil {
				log.Fatalf("Error writing protobuf: %v", err)
			}
		}
		if err := rd.Close(); err != nil {
			log.Printf("Error closing result reader: %v", err)
		}
		fileStore.ClearData()
	}

	os.Exit(errors)
}

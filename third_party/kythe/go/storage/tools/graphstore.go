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

// Binary graphstore is the frontend to Kythe's service. graphstore
// exposes two interfaces to the underlying GraphStore: a K-RPC HTTP server
// (specified by a --port) and a K-RPC server request stream piped into os.Stdin
// (specified by --streaming). Both implement the /GraphStore/{Read,Write,Scan}
// methods of a storage.GraphStore and can be run concurrently.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"third_party/kythe/go/rpc/server"
	"third_party/kythe/go/storage"
	"third_party/kythe/go/storage/client"
	"third_party/kythe/go/storage/gsutil"
	"third_party/kythe/go/storage/service"
)

var (
	gs storage.GraphStore

	// Frontends
	port            = flag.Int("port", 0, "Port for /GraphStore/{Read,Write,Scan} K-RPC interface; if 0, don't start server")
	streamingServer = flag.Bool("streaming", false, "Use stdin/stdout for a K-RPC streaming server")

	remotes = flag.String("remotes", "", "Comma-separated list of GraphStore address to send copies of all requests")
)

func init() {
	gsutil.Flag(&gs, "graphstore", "Path to leveldb/kv database")
}

func main() {
	flag.Parse()
	log.SetPrefix("graphstore: ")

	if !*streamingServer && *port <= 0 {
		log.Fatal("Must specify --port or --streaming")
	}

	if gs == nil {
		log.Fatal("No --graphstore specified!")
	}
	defer gsutil.LogClose(gs)
	gsutil.EnsureGracefulExit(gs)

	addrs := strings.Split(*remotes, ",")
	if *remotes != "" && len(addrs) > 0 {
		stores := make([]storage.GraphStore, len(addrs)+1)
		for idx, addr := range addrs {
			log.Printf("Connecting to remote %q", addr)
			stores[idx] = client.New(addr)
			defer gsutil.LogClose(stores[idx])
		}
		stores[len(addrs)] = gs
		gs = storage.NewProxy(stores...)
	}

	s, err := service.New(gs)
	if err != nil {
		log.Fatal(err)
	}
	endpoint := server.Endpoint{s}

	// Handle os.Stdin requests/entries
	if *streamingServer {
		if *port > 0 {
			// Run in background; block on HTTP server instead
			go runStreamingServer(endpoint, os.Stdin, os.Stdout)
		} else {
			runStreamingServer(endpoint, os.Stdin, os.Stdout)
		}
	}

	// Launch HTTP server
	if *port > 0 {
		http.Handle("/", endpoint)
		log.Println("HTTP Server launched")
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}
}

func runStreamingServer(endpoint server.Endpoint, in io.Reader, out io.WriteCloser) {
	if err := endpoint.ServePipes(server.Map(nil), in, out); err != nil {
		log.Printf("streaming server error: %v", err)
	}
	out.Close()
}

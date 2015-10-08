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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/shipshape/shipshape/service"
	"github.com/google/shipshape/shipshape/util/rpc/server"

	rpcpb "github.com/google/shipshape/shipshape/proto/shipshape_rpc_proto"
)

var (
	servicePort = flag.Int("port", 10007, "Service port")
	// TODO(supertri): add a stringList flag option
	analyzers    = flag.String("analyzer_services", "localhost:10005,localhost:10006,localhost:10008", "Addresses of analyzer services (comma-separated)")
	startService = flag.Bool("start_service", false, "Start a shipshape service, if false we use streams to handle requests (stdin/stdout)")
)

const (
	serviceName = "ShipshapeService"
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	analyzerList := strings.Split(*analyzers, ",")

	log.Printf("Waiting for analyzers to become healthy...")
	healthErrors := service.WaitForAnalyzers(analyzerList)
	allHealthy := true
	for addr, err := range healthErrors {
		if err != nil {
			log.Printf("Analyzer at %s failed to become healthy: %v", addr, err)
			allHealthy = false
		} else {
			log.Printf("Analyzer at %s registered", addr)
		}
	}
	if allHealthy {
		log.Printf("All analyzers deemed healthy")
	}

	shipshapeService := service.NewDriver(analyzerList)

	if *startService {
		// Start shipshape service
		s1 := server.Service{Name: serviceName}
		if err := s1.Register(shipshapeService); err != nil {
			log.Fatalf("Registering shipshape service failed: %v", err)
		}
		addr := fmt.Sprintf(":%d", *servicePort)
		log.Printf("Starting server endpoint at %q with service name %s\n", addr, serviceName)
		http.Handle("/", server.Endpoint{&s1})
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Server startup failed: %v", err)
		}
	} else {
		log.Println("Waiting for stdin. Specify --start_service if you meant to start as a service.")

		// Read request bytes from stdin
		requestBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal("failed to read stdin: ", err)
		}

		log.Printf("Read shipshape request on stdin [%v bytes]", len(requestBytes))

		// Convert bytes from stdin to Shipshape request
		request := new(rpcpb.ShipshapeRequest)
		err = proto.Unmarshal(requestBytes, request)
		if err != nil {
			log.Fatal("failed to unmarshal shipshape request stream: ", err)
		}

		c := make(chan *rpcpb.ShipshapeResponse)

		go func() {
			if err := shipshapeService.Run(nil, request, c); err != nil {
				log.Printf("Failed to run on server: %v", err)
			}
		}()

		log.Print("Sent request to driver")

		response := <-c

		log.Printf("Shipshape response: [%s]", response)

		responseBytes, err := proto.Marshal(response)
		if err != nil {
			log.Fatal("failed to marshal shipshape response: ", err)
		}

		log.Printf("Writing Shipshape response to stdout")

		os.Stdout.Write(responseBytes)
	}
}

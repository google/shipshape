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

// Binary android_lint_service implements the Shipshape analyzer service. It runs the
// androidlint analyzer on files that it is given. It only runs on java and xml files
// within an Android project.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/shipshape/shipshape/api"
	"shipshape/androidlint_analyzer/androidlint"
	"shipshape/util/rpc/server"

	ctxpb "github.com/google/shipshape/shipshape/proto/shipshape_context_proto"
)

var (
	servicePort = flag.Int("port", 10005, "Service port")
)

func main() {
	flag.Parse()

	// The shipshape service will connect to an AnalyzerService
	// at port 10005 in the container. (The service will map this to a different external
	// port at startup so that it doesn't clash with other analyzers.)
	s := server.Service{Name: "AnalyzerService"}
	as := api.CreateAnalyzerService([]api.Analyzer{new(androidlint.Analyzer)}, ctxpb.Stage_PRE_BUILD)
	if err := s.Register(as); err != nil {
		log.Fatalf("Registering analyzer service failed: %v", err)
	}

	addr := fmt.Sprintf(":%d", *servicePort)
	fmt.Fprintf(os.Stderr, "-- Starting server endpoint at %q\n", addr)
	http.Handle("/", server.Endpoint{&s})

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

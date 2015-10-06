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
	"log"
	"net/http"

	"github.com/google/shipshape/shipshape/api"
	"shipshape/analyzers/codealert"
	"shipshape/analyzers/govet"
	"shipshape/analyzers/jshint"
	"shipshape/analyzers/postmessage"
	"shipshape/analyzers/pylint"
	"shipshape/analyzers/wordcount"
	"shipshape/util/rpc/server"

	ctxpb "shipshape/proto/shipshape_context_proto"
)

var (
	servicePort = flag.Int("port", 10005, "Service port")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	analyzers := []api.Analyzer{
		new(postmessage.PostMessageAnalyzer),
		new(wordcount.WordCountAnalyzer),
		new(jshint.JSHintAnalyzer),
		new(codealert.CodeAlertAnalyzer),
		new(pylint.PyLintAnalyzer),
		new(govet.GoVetAnalyzer),
	}
	analyzerService := api.CreateAnalyzerService(analyzers, ctxpb.Stage_PRE_BUILD)

	s1 := server.Service{Name: "AnalyzerService"}
	if err := s1.Register(analyzerService); err != nil {
		log.Fatalf("Registering analyzer service failed: %v", err)
	}

	addr := fmt.Sprintf(":%d", *servicePort)

	log.Printf("-- Starting server endpoint at %q", addr)

	http.Handle("/", server.Endpoint{&s1})

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

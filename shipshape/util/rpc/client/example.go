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

// Binary example provides a CLI client to the K-RPC example server.
//
// Usage:
//   ./server/example --port 10005 &
//   ./client/example --server_address :10005 /Echo/Entry '{"source": {"signature": "sig", "language": "test"}}'
//   ./client/example --server_address :10005 /Echo/Repeat '{"count": 5, "label": "Some message here"}'
//   kill $(lsof -ti :10005 -s TCP:LISTEN)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/google/shipshape/shipshape/util/rpc/client"
)

var (
	address      = flag.String("server_address", ":10005", "Address of K-RPC example server")
	waitDuration = flag.Duration("readiness_timeout", 10*time.Second, "Time to wait until server is ready")
	legacyCall   = flag.Bool("legacy", false, "Use legacy JSON-RPC call semantics")
)

func main() {
	flag.Parse()

	c := client.NewHTTPClient(*address)
	if err := c.WaitUntilReady(*waitDuration); err != nil {
		log.Fatal(err)
	}

	serviceMethod, params := flag.Arg(0), json.RawMessage(flag.Arg(1))
	if *legacyCall {
		var resp json.RawMessage
		if err := c.Call(serviceMethod, &params, &resp); err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(resp))
	} else {
		rd := c.Stream(serviceMethod, &params)
		defer rd.Close()
		for {
			var resp json.RawMessage
			if err := rd.NextResult(&resp); err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(resp))
		}
	}
}

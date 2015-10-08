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

// Binary unwrap_results reads in a JSON K-RPC response stream and writes each
// result.  On an error, the processing halts and the binary exits with an
// error.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/shipshape/shipshape/util/rpc/client"
)

func main() {
	in := os.Stdin
	out := os.Stdout

	rd := client.NewPipeReader(in)
	var result json.RawMessage
	err := rd.Receive(&result, func(id []byte, err error, end bool) bool {
		if err != nil {
			log.Fatalf("Error in response stream: %v", err)
		}

		if !end {
			fmt.Fprintln(out, string(result))
		}
		return true
	})
	if err != nil {
		log.Fatalf("RPC error: %v", err)
	}
}

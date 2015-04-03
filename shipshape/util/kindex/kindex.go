/*
 * Copyright 2015 Google Inc. All rights reserved.
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

// Binary kindex is a simple utility to print out the contents of a kindex file in a more
// human readable form. This is sometimes useful when debugging the output from kythe.
package main

import (
	"flag"
	"fmt"
	kythe_index "third_party/kythe/go/platform/kindex"
)

var (
	file = flag.String("kindex", "", "kindex file to be parsed and printed")
)

func main() {
	flag.Parse()
	info, err := kythe_index.Open(*file)
	if err != nil {
		fmt.Printf("Could not open kindex file: %v", err)
		return
	}
	fmt.Println("File parsed")
	fmt.Printf("%v", info)
}

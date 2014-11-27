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

// Binary foreach reads a delimited stream on stdin and calls out to external
// process for each delimited value, passing it into the process's stdin.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"third_party/kythe/go/platform/delimited"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s cmd arg...\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(2)
	}
}

var printCount = flag.Bool("count", false, "Print count of records at EOF")

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
	}

	binary, args := flag.Arg(0), flag.Args()[1:]

	rd := delimited.NewReader(os.Stdin)
	count := 0
	for {
		rec, err := rd.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		count++

		cmd := exec.Command(binary, args...)
		cmd.Stdin = bytes.NewBuffer(rec)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
	}

	if *printCount {
		fmt.Println(count)
	}
}

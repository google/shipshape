// Binary viewindex prints a .kindex as JSON to stdout.
//
// Example:
//   viewindex compilation.kindex | jq .
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"third_party/kythe/go/platform/indexinfo"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <kindex-file>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

var printFiles = flag.Bool("files", false, "Print file contents as well as the compilation")

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	path := flag.Arg(0)
	idx, err := indexinfo.Open(path)
	if err != nil {
		log.Fatalf("Error reading %q: %v", path, err)
	}

	en := json.NewEncoder(os.Stdout)
	if *printFiles {
		if err := en.Encode(idx); err != nil {
			log.Fatalf("Error encoding JSON: %v", err)
		}
	} else {
		if err := en.Encode(idx.Compilation); err != nil {
			log.Fatalf("Error encoding JSON compilation: %v", err)
		}
	}
}

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

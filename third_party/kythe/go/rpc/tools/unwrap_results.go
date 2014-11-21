// Binary unwrap_results reads in a JSON K-RPC response stream and writes each
// result.  On an error, the processing halts and the binary exits with an
// error.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"third_party/kythe/go/rpc/client"
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

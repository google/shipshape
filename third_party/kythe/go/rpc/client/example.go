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

	"third_party/kythe/go/rpc/client"
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

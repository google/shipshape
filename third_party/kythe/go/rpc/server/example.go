// Binary example is a simple example KRPC server to demonstrate how to use the
// krpc package.  The server exposes a simple "echo" service that copies
// modified versions of its inputs back to its output, and an "error" service
// that generates RPC errors.
//
// Usage:
//   example -port 10005 -echo MyLittleService
//
// Fetch a service listing:
//   curl -X POST -d '{"jsonrpc": "2.0", "id": 1, "method": "/ServerInfo/List"}' localhost:10005
//
// Construct an Entry:
//  curl -X POST -d '{"jsonrpc": "2.0",
//                    "method": "/MyLittleService/Entry",
//                    "id": 2,
//                    "params": {"source": {"signature": "123456"}}}' localhost:10005
//
// Echo a message multiple times:
//   curl -X POST -d '{"jsonrpc": "2.0 streaming",
//                     "method": "/MyLittleService/Repeat",
//                     "id": 3,
//                     "params": {"count": 5, "label": "five"}}' localhost:10005
//   curl -X POST -d '{"jsonrpc": "2.0",
//                     "method": "/MyLittleService/Repeat",
//                     "id": 4,
//                     "params": {"count": 5, "label": "five"}}' localhost:10005
//
// Generate an RPC error:
//   curl -X POST -d '{"jsonrpc": "2.0",
//                     "method": "/Error/Fail",
//                     "id": 5,
//                     "params": {"Str": "error string"}}' localhost:10005
//
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"third_party/kythe/go/rpc/server"

	spb "third_party/kythe/proto/storage_proto"
)

var (
	echoService  = flag.String("echo", "", `Echo service name ("" to use service type name, "Echo"`)
	errorService = flag.String("error", "", `Error service name ("" to use service type name, "Error"`)
	printSpec    = flag.Bool("spec", false, "Print the server spec at startup")
	servicePort  = flag.Int("port", 10005, "Service port")
)

// Echo exposes methods that copy their input to their output.
type Echo struct{}

// Entry outputs an Entry based on the ReadRequest's Source; returns an error if
// the ReadRequest does not contain a Source.
func (Echo) Entry(ctx server.Context, in *spb.ReadRequest) (*spb.Entry, error) {
	if in.Source != nil {
		name := "/kythe/node/kind"
		return &spb.Entry{Source: in.Source, FactName: &name, FactValue: []byte("test")}, nil
	}
	return nil, errors.New("empty source")
}

// RepeatReq is the input message for Echo.Repeat.
type RepeatReq struct {
	Count int    `json:"count"`
	Label string `json:"label"`
}

// Repeat writes Count copies of Label to the output.
func (Echo) Repeat(ctx server.Context, in *RepeatReq, out chan<- string) error {
	if in.Count < 0 {
		return errors.New("count out of range")
	}
	for i := 0; i < in.Count; i++ {
		out <- fmt.Sprintf("(%d) %s", i+1, in.Label)
	}
	return nil
}

// Error exposes a single method that always returns an error.
type Error struct{}

// An ErrorRequest asks the server to return an error based on a string
type ErrorRequest struct {
	Str string
}

// Fail returns its input (a string) as an error from the RPC.
func (Error) Fail(ctx server.Context, in *ErrorRequest) (string, error) {
	// the return result is thrown away on an error
	return in.Str, errors.New(in.Str)
}

func main() {
	flag.Parse()

	// Register the echo service.  If the service name is empty, the Register
	// method will populate it with the name of the type being registered.
	s1 := server.Service{Name: *echoService}
	if err := s1.Register(Echo{}); err != nil {
		log.Fatalf("Registering echo service failed: %v", err)
	}

	// Register the error service.
	s2 := server.Service{Name: *errorService}
	if err := s2.Register(Error{}); err != nil {
		log.Fatalf("Registering error service failed: %v", err)
	}

	// If requested, print the service specifications.
	if *printSpec {
		fmt.Fprintf(os.Stderr, "-- Server spec for %q:\n%v\n", s1.Name, mustSpec(&s1))
		fmt.Fprintf(os.Stderr, "-- Server spec for %q:\n%v\n", s2.Name, mustSpec(&s2))
	}

	addr := fmt.Sprintf(":%d", *servicePort)
	fmt.Fprintf(os.Stderr, "-- Starting server endpoint at %q\n", addr)
	http.Handle("/", server.Endpoint{&s1, &s2})

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func mustSpec(s *server.Service) string {
	spec, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		log.Fatalf("Marshaling to JSON failed: %v", err)
	}
	return string(spec)
}

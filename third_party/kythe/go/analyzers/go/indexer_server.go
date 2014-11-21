// Program goindexer provides a CompilationAnalyzer service on the specified port
// to produce Kythe index artifacts for Go compilation units.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"third_party/kythe/go/analyzers/go/grok"
	"third_party/kythe/go/platform/cache"
	"third_party/kythe/go/platform/service"
	"third_party/kythe/go/rpc/server"
	"third_party/kythe/go/storage/files/client"
)

var (
	defaultFDS = flag.String("default_fds", "",
		"The address of a FileDataService to use if none is given in the request")
	port      = flag.Int("port", 0, "Service port (required)")
	cacheSize = byteSize("file_cache_size", 4<<20,
		"Size of in-memory file cache, in bytes (units OK)")
)

func byteSize(name string, init int, desc string) *int {
	b := cache.ByteSize(init)
	flag.Var(&b, name, desc)
	return (*int)(&b)
}

func main() {
	flag.Parse()

	if *port <= 0 {
		log.Fatal("You must specify a non-zero value for --port")
	}

	indexer := grok.NewIndexer()
	indexer.EmitEntries = true
	goIndexer := &service.Service{
		Analyzer: indexer,
		Cache:    cache.New(*cacheSize),
	}

	if *defaultFDS != "" {
		goIndexer.DefaultFD = client.New(*defaultFDS)
		log.Printf("Using FileDataService at %q.", *defaultFDS)
	} else {
		log.Printf("No default FileDataService address was provided.")
	}

	s := server.Service{Name: "CompilationAnalyzer"}
	if err := s.Register(goIndexer); err != nil {
		log.Fatalf("could not register CompilationAnalyzer Service: %v", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server endpoint at %q\n", addr)
	http.Handle("/", server.Endpoint{&s})

	host, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	fdsAddress := fmt.Sprintf("%s:%d", host, *port)
	log.Printf("CompilationAnalyzer address: %s", fdsAddress)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

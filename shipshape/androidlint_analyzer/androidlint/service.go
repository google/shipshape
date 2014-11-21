// Binary android_lint_service implements the Shipshape analyzer service. It runs the
// androidlint analyzer on files that it is given. It only runs on java and xml files
// within an Android project.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"third_party/kythe/go/rpc/server"
	"shipshape/androidlint_analyzer/androidlint"
	"shipshape/api"
)

var (
	servicePort = flag.Int("port", 10005, "Service port")
)

func main() {
	flag.Parse()

	s := server.Service{Name: "AnalyzerService"}
	as := api.CreateAnalyzerService([]api.Analyzer{new(androidlint.Analyzer)})
	if err := s.Register(as); err != nil {
		log.Fatalf("Registering analyzer service failed: %v", err)
	}

	addr := fmt.Sprintf(":%d", *servicePort)
	fmt.Fprintf(os.Stderr, "-- Starting server endpoint at %q\n", addr)
	http.Handle("/", server.Endpoint{&s})

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

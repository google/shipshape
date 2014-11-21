package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"third_party/kythe/go/rpc/server"
	"shipshape/analyzers/codealert"
	"shipshape/analyzers/govet"
	"shipshape/analyzers/jshint"
	"shipshape/analyzers/postmessage"
	"shipshape/analyzers/pylint"
	"shipshape/analyzers/wordcount"
	"shipshape/api"
)

var (
	servicePort = flag.Int("port", 10005, "Service port")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	analyzers := []api.Analyzer{
		new(postmessage.PostMessageAnalyzer),
		new(wordcount.WordCountAnalyzer),
		new(jshint.JSHintAnalyzer),
		new(codealert.CodeAlertAnalyzer),
		new(pylint.PyLintAnalyzer),
		new(govet.GoVetAnalyzer),
	}
	analyzerService := api.CreateAnalyzerService(analyzers)

	s1 := server.Service{Name: "AnalyzerService"}
	if err := s1.Register(analyzerService); err != nil {
		log.Fatalf("Registering analyzer service failed: %v", err)
	}

	addr := fmt.Sprintf(":%d", *servicePort)

	log.Printf("-- Starting server endpoint at %q", addr)

	http.Handle("/", server.Endpoint{&s1})

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

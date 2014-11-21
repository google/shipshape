// Binary triples implements a converter from an Entry stream to a stream of triples.
//
// Examples:
//   triples < entries > triples.nq
//   triples entries > triples.nq.gz
//   triples entries triples.nq
//   curl --compressed -X POST \
//     -d '{"jsonrpc": "2.0 streaming", "id": 1, "method": "/GraphStore/Scan"}' localhost:1222 | \
//     unwrap_results | entrystream --read_json | triples | gzip > gs.nq.gz
//
// Reference: http://en.wikipedia.org/wiki/N-Triples
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"third_party/kythe/go/storage/stream"
	"third_party/kythe/go/util/kytheuri"
	"third_party/kythe/go/util/schema"

	spb "third_party/kythe/proto/storage_proto"
)

var (
	keepReverseEdges = flag.Bool("keep_reverse_edges", false, "Do not filter reverse edges from triples output")
	quiet            = flag.Bool("quiet", false, "Do not emit logging messages")
)

func main() {
	flag.Parse()

	in := os.Stdin
	if len(flag.Args()) > 0 {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatalf("Failed to open input file %q: %v", flag.Arg(0), err)
		}
		defer file.Close()
		in = file
	}

	out := os.Stdout
	if len(flag.Args()) > 1 {
		file, err := os.Create(flag.Arg(1))
		if err != nil {
			log.Fatalf("Failed to create output file %q: %v", flag.Arg(1), err)
		}
		defer file.Close()
		out = file
	}

	var (
		reverseEdges int
		triples      int
	)
	for entry := range stream.ReadEntries(in) {
		if schema.EdgeDirection(entry.GetEdgeKind()) == schema.Reverse && !*keepReverseEdges {
			reverseEdges++
			continue
		}

		t, err := toTriple(entry)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintln(out, t)
		triples++
	}

	if !*quiet {
		if !*keepReverseEdges {
			log.Printf("Skipped %d reverse edges", reverseEdges)
		}
		log.Printf("Wrote %d triples", triples)
	}
}

type triple struct {
	Subject   string
	Predicate string
	Object    string
}

func (t *triple) String() string {
	return fmt.Sprintf("%q %q %q .", t.Subject, t.Predicate, t.Object)
}

// toTriple converts an Entry to the triple file format. Returns an error if the
// entry is not valid.
func toTriple(entry *spb.Entry) (*triple, error) {
	if entry.Source == nil || (entry.Target == nil) != (entry.GetEdgeKind() == "") {
		return nil, fmt.Errorf("invalid entry: %v", entry)
	}

	t := &triple{
		Subject: kytheuri.FromVName(entry.Source).String(),
	}
	if entry.GetEdgeKind() != "" {
		t.Predicate = entry.GetEdgeKind()
		t.Object = kytheuri.FromVName(entry.Target).String()
	} else {
		t.Predicate = entry.GetFactName()
		t.Object = string(entry.GetFactValue())
	}
	return t, nil
}

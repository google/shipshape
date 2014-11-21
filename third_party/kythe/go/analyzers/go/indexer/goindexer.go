// Goindexer binary implements a Go indexer for one specific index file.
// Receives an index file path as an argument and returns the list of
// output entries to the stdout.
// Usage: goindexer index_file_path
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"third_party/kythe/go/analyzers/go/grok"
	"third_party/kythe/go/platform/indexinfo"
	"third_party/kythe/go/rpc/stream"
	"third_party/kythe/go/storage/files"

	apb "third_party/kythe/proto/analysis_proto"
)

// MemoryStore implements an in memory store using the files.InMemoryStore
// implementation. Additionally, it implements the analysis.Fetch interface.
type MemoryStore struct {
	files.InMemoryStore
}

// Fetch returns the content of the file specified by path and its digest.
// Returns an error, if any.
func (p *MemoryStore) Fetch(path, digest string) ([]byte, error) {
	return p.InMemoryStore.FileData(path, digest)
}

// FileSink implements the analysis.Sink interface for a file descriptor.
type FileSink struct {
	stream.Writer
}

func NewFileSink(file *os.File) *FileSink {
	return &FileSink{
		Writer: stream.NewWriter(file, false),
	}
}

// WriteBytes writes the input bytes to the file. Returns an error, if any.
func (p *FileSink) WriteBytes(bytes []byte) error {
	return p.Writer.Put(bytes)
}

func main() {
	flag.Parse()

	// Accept only one index file as input.
	if (len(flag.Args())) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: goindexer index_file_path")
		log.Fatalf("Go indexer must only receive 1 argument; got %d", len(flag.Args()))
	}

	// Create an in memory file store.
	memStore := &MemoryStore{
		InMemoryStore: files.InMemory(),
	}

	// The path to the .kindex file we are interested in.
	indexFile := flag.Args()[0]

	// Open the index file.
	idx, err := indexinfo.Open(indexFile)
	if err != nil {
		log.Fatalf("Error opening indexinfo: %v", err)
	}

	// Add the contained files in the file store.
	if err := memStore.InMemoryStore.AddData(idx.Files...); err != nil {
		log.Fatalf("Error adding files to file data service: %v", err)
	}

	req := &apb.AnalysisRequest{
		Compilation: idx.Compilation,
	}

	// Create a new indexer.
	indexer := grok.NewIndexer()
	indexer.EmitEntries = true

	// Create a file sink pointing to stdout.
	stdoutSink := NewFileSink(os.Stdout)

	// Run the analyzer; read the contents from the in memory store, output to stdout.
	indexer.Analyze(req, memStore, stdoutSink)
}

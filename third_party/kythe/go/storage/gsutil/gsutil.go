// Package gsutil is collection of helper functions for storage tools.
package gsutil

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"third_party/kythe/go/storage"
	"third_party/kythe/go/storage/client"
	"third_party/kythe/go/storage/kv"
	"third_party/kythe/go/storage/leveldb"
)

type gsFlag struct {
	gs *storage.GraphStore
}

// String implements part of the flag.Value interface.
func (f *gsFlag) String() string { return fmt.Sprintf("%T", *f.gs) }

// Set implements part of the flag.Value interface.
func (f *gsFlag) Set(str string) (err error) {
	*f.gs, err = parseGraphStore(str)
	return
}

// Flag defines a GraphStore flag with the specified name and usage string.
func Flag(gs *storage.GraphStore, name, usage string) {
	if gs == nil {
		log.Fatalf("GraphStoreFlag given nil GraphStore pointer")
	}
	f := gsFlag{gs: gs}
	flag.Var(&f, name, usage)
}

// parseGraphStore returns a GraphStore for the given specification.
func parseGraphStore(str string) (storage.GraphStore, error) {
	str = strings.TrimSpace(str)
	switch {
	case str == "in-memory":
		gs, err := kv.InMemory()
		if err != nil {
			return nil, err
		}
		return gs, nil
	case client.ValidAddr(str):
		return client.New(str), nil
	case leveldb.ValidDB(str):
		gs, err := leveldb.Open(str)
		if err != nil {
			return nil, fmt.Errorf("error opening LevelDB GraphStore: %v", err)
		}
		return gs, nil
	case kv.ValidDB(str):
		gs, err := kv.Open(str)
		if err != nil {
			return nil, fmt.Errorf("error opening kv GraphStore: %v", err)
		}
		return gs, nil
	default:
		return nil, fmt.Errorf("unknown GraphStore: %q", str)
	}
}

// EnsureGracefulExit will try to close each gs when notified of an Interrupt,
// SIGTERM, or Kill signal and immediately exit the program unsuccessfully. Any
// errors will be logged. This function should only be called once and closing
// the GraphStores manually is still needed when the program does not receive a
// signal to quit.
func EnsureGracefulExit(gs ...storage.GraphStore) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, os.Kill)
	go func() {
		sig := <-c
		log.Printf("graphstore: signal %v", sig)
		for _, g := range gs {
			LogClose(g)
		}
		os.Exit(1)
	}()
}

// LogClose closes gs and logs any resulting error.
func LogClose(gs storage.GraphStore) {
	if err := gs.Close(); err != nil {
		log.Printf("GraphStore failed to close: %v", err)
	}
}

// UsageError prints msg to stderr, calls flag.Usage, and exits the program
// unsuccessfully.
func UsageError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	flag.Usage()
	os.Exit(1)
}

// UsageErrorf prints str formatted with the given vals to stderr, calls
// flag.Usage, and exits the program unsuccessfully.
func UsageErrorf(str string, vals ...interface{}) {
	UsageError(fmt.Sprintf(str, vals...))
}

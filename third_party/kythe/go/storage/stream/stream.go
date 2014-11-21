// Package stream provides utility functions to consume Entry streams.
package stream

import (
	"encoding/json"
	"io"
	"log"

	"third_party/kythe/go/platform/delimited"

	spb "third_party/kythe/proto/storage_proto"
)

// ReadEntries reads a stream of Entry protobufs from r.
func ReadEntries(r io.Reader) <-chan *spb.Entry {
	ch := make(chan *spb.Entry)
	go func() {
		defer close(ch)
		rd := delimited.NewReader(r)
		for {
			var entry spb.Entry
			if err := rd.NextProto(&entry); err == io.EOF {
				break
			} else if err != nil {
				log.Fatalf("Error decoding Entry: %v", err)
			}
			ch <- &entry
		}
	}()
	return ch
}

// ReadJSONEntries reads a JSON stream of Entry protobufs from r.
func ReadJSONEntries(r io.Reader) <-chan *spb.Entry {
	ch := make(chan *spb.Entry)
	go func() {
		defer close(ch)
		de := json.NewDecoder(r)
		for {
			var entry spb.Entry
			if err := de.Decode(&entry); err == io.EOF {
				break
			} else if err != nil {
				log.Fatalf("Error decoding Entry: %v", err)
			}
			ch <- &entry
		}
	}()
	return ch
}

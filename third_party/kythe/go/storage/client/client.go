// Package client provides a storage.GraphStore interface implementation for a
// remote K-RPC GraphStore service.
package client

import (
	"io"
	"third_party/kythe/go/rpc/client"
	"third_party/kythe/go/storage"

	"code.google.com/p/goprotobuf/proto"

	spb "third_party/kythe/proto/storage_proto"
)

type remoteGraphStore struct {
	c *client.Client
}

// ValidAddr returns true if the given string is a valid GraphStore address
// (not necessarily reachable).
func ValidAddr(addr string) bool { return client.ValidHTTPAddr(addr) }

// New returns a storage.GraphStore that forwards Reads, Writes, and Scans to
// GraphStore service located at addr.
func New(addr string) storage.GraphStore {
	return &remoteGraphStore{client.NewHTTPClient(addr)}
}

// remoteCall calls the remote GraphStore serviceMethod with the req argument.
func (p *remoteGraphStore) remoteCall(serviceMethod string, req proto.Message, entries chan<- *spb.Entry) error {
	rd := p.c.Stream(serviceMethod, req)
	defer rd.Close()
	for {
		var entry spb.Entry
		if err := rd.NextResult(&entry); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		entries <- &entry
	}
}

// Read implements a storage.GraphStore and forwards the ReadRequest to the
// remote GraphStores.
func (p *remoteGraphStore) Read(req *spb.ReadRequest, stream chan<- *spb.Entry) error {
	return p.remoteCall("/GraphStore/Read", req, stream)
}

// Scan implements a storage.GraphStore and forwards the ReadRequest to the
// remote GraphStores.
func (p *remoteGraphStore) Scan(req *spb.ScanRequest, stream chan<- *spb.Entry) error {
	return p.remoteCall("/GraphStore/Scan", req, stream)
}

// Write implements a storage.GraphStore and forwards the ReadRequest to the
// remote GraphStores.
func (p *remoteGraphStore) Write(req *spb.WriteRequest) error {
	var v interface{} // throw result away
	return p.c.Call("/GraphStore/Write", req, &v)
}

// Close implements a storage.GraphStore. Does nothing.
func (*remoteGraphStore) Close() error { return nil }

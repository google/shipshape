// Package service exposes a GraphStore as a K-RPC service.
package service

import (
	"fmt"

	"third_party/kythe/go/rpc/server"
	"third_party/kythe/go/storage"

	spb "third_party/kythe/proto/storage_proto"
)

// A graphStoreService is a wrapper around a storage.GraphStore that can be
// registered by a server.Service.
type graphStoreService struct {
	s storage.GraphStore
}

// New returns a new RPC service for a GraphStore.
func New(g storage.GraphStore) (*server.Service, error) {
	s := &server.Service{Name: "GraphStore"}
	if err := s.Register(&graphStoreService{g}); err != nil {
		return nil, fmt.Errorf("could not register GraphStore service: %v", err)
	}
	return s, nil
}

// Read throws away ctx and delegates to the Read method of the underlying storage.GraphStore
func (s *graphStoreService) Read(ctx server.Context, req *spb.ReadRequest, out chan<- *spb.Entry) error {
	return s.s.Read(req, out)
}

// Scan throws away ctx and delegates to the Scan method of the underlying storage.GraphStore
func (s *graphStoreService) Scan(ctx server.Context, req *spb.ScanRequest, out chan<- *spb.Entry) error {
	return s.s.Scan(req, out)
}

// Write throws away ctx and delegates to the Write method of the underlying storage.GraphStore
func (s *graphStoreService) Write(ctx server.Context, req *spb.WriteRequest) (struct{}, error) {
	return struct{}{}, s.s.Write(req)
}

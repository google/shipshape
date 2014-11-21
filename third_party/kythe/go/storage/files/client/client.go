// Package client provides a service.FileStore interface implementation for a
// remote K-RPC FileStore service.
package client

import (
	"fmt"
	"io"

	"third_party/kythe/go/rpc/client"

	"code.google.com/p/goprotobuf/proto"

	apb "third_party/kythe/proto/analysis_proto"
)

// A FileData is a K-RPC client wrapper for a files.FileStore.
type FileData struct {
	c *client.Client
}

// New returns a FileData that forwards FileData calls to
// files.FileStore service located at addr.
func New(addr string) *FileData {
	return &FileData{client.NewHTTPClient(addr)}
}

// remoteCall calls the remote FileStore serviceMethod with the req argument.
func (p *FileData) remoteCall(serviceMethod string, req proto.Message, out chan<- *apb.FileData) error {
	defer close(out)

	rd := p.c.Stream(serviceMethod, req)
	defer rd.Close()
	for {
		var fileData apb.FileData
		if err := rd.NextResult(&fileData); err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("filedata: remote call error: %v", err)
		}
		out <- &fileData
	}
}

// get implements the client-side of the FileData request. It forwards the FileInfo to the
//remote FileDataService. Returns the results on the stream channel.
func (p *FileData) get(info *apb.FileInfo, stream chan<- *apb.FileData) error {
	return p.remoteCall("/FileData/Get", info, stream)
}

// FileData implements a files.FileStore and forwards the request to the remote FileDataService.
func (p *FileData) FileData(path, digest string) ([]byte, error) {
	fileInfo := &apb.FileInfo{
		Path:   &path,
		Digest: &digest,
	}

	stream := make(chan *apb.FileData, 1)

	if err := p.get(fileInfo, stream); err != nil {
		return nil, err
	}

	fileData := <-stream
	return fileData.GetContent(), nil
}

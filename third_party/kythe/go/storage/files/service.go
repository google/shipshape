package files

import (
	"fmt"

	"third_party/kythe/go/rpc/server"

	apb "third_party/kythe/proto/analysis_proto"
)

// FileDataService is a K-RPC service wrapper for a files.FileStore.
type FileDataService struct {
	Store FileStore
}

// NewFileDataService returns a new FileDataService wrapping a FileStore.
func NewFileDataService(store FileStore) *FileDataService {
	return &FileDataService{Store: store}
}

// Get passes the FileInfo's metadata to the underlying files.FileStore to retrieve the file's data
func (fds *FileDataService) Get(ctx server.Context, info *apb.FileInfo, out chan<- *apb.FileData) error {
	data, err := fds.Store.FileData(info.GetPath(), info.GetDigest())
	if err != nil {
		return fmt.Errorf("unable to get %v: %v", info, err)
	}
	out <- &apb.FileData{Info: info, Content: data}
	return nil
}

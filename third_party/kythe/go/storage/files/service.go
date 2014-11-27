/*
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

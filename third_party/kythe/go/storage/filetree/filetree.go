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

// Package filetree implements an in-memory FileTree and Directory model used to
// store VName-based file paths.
package filetree

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"third_party/kythe/go/storage"

	spb "third_party/kythe/proto/storage_proto"

	"code.google.com/p/goprotobuf/proto"
)

const (
	nodeKindLabel = "/kythe/node/kind"
	fileNodeKind  = "file"
)

// A FileTree is a group of corpora, their respective roots, and their top-level
// Directory.
type FileTree struct {
	// corpus -> root -> top-level Directory
	CorporaRoots map[string]map[string]*Directory
}

// A Directory is a named container for other Directories and VName-based files.
type Directory struct {
	// Basename of the directory
	Name string `json:"name"`

	// Map of sub-directories key'd by their basename
	Dirs map[string]*Directory `json:"dirs"`

	// Map of files within the Directory key'd by their basename.  Note: the value
	// is a slice because there may be different versions of file's signature
	// (its digest)
	Files map[string][]*spb.VName `json:"files"`
}

// NewTree returns an empty FileTree
func NewTree() *FileTree {
	return &FileTree{make(map[string]map[string]*Directory)}
}

// Populate adds each file node in the GraphStore to the FileTree
func (t *FileTree) Populate(gs storage.GraphStore) error {
	entries := make(chan *spb.Entry)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range entries {
			if entry.GetFactName() == nodeKindLabel && string(entry.GetFactValue()) == fileNodeKind {
				t.AddFile(entry.Source)
			}
		}
	}()
	err := gs.Scan(&spb.ScanRequest{FactPrefix: proto.String(nodeKindLabel)}, entries)
	close(entries)
	if err != nil {
		return fmt.Errorf("failed to Scan GraphStore for directory structure: %v", err)
	}
	wg.Wait()
	return nil
}

// AddFile adds the given VName file to the FileTree.
func (t *FileTree) AddFile(file *spb.VName) {
	root := t.ensureCorpusRoot(file.GetCorpus(), file.GetRoot())
	root.addFile(file)
}

// LookupDir returns the Directory for the given corpus/root/path. nil is
// returned if one is not found.
func (t *FileTree) LookupDir(corpus, root, path string) *Directory {
	roots := t.CorporaRoots[corpus]
	if roots == nil {
		return nil
	}
	dir := roots[root]
	if dir == nil {
		return nil
	} else if path == "/" {
		return dir
	}
	for _, component := range strings.Split(strings.Trim(path, "/"), "/") {
		dir = dir.Dirs[component]
		if dir == nil {
			return nil
		}
	}
	return dir
}

// LookupFiles returns a slice of VName files residing in the given
// corpus/root and with the given path.
func (t *FileTree) LookupFiles(corpus, root, path string) []*spb.VName {
	dir := t.LookupDir(corpus, root, filepath.Dir(path))
	if dir == nil {
		return nil
	}
	return dir.Files[filepath.Base(path)]
}

func (t *FileTree) ensureCorpusRoot(corpus, root string) *Directory {
	roots := t.CorporaRoots[corpus]
	if roots == nil {
		roots = make(map[string]*Directory)
		t.CorporaRoots[corpus] = roots
	}

	dir := roots[root]
	if dir == nil {
		dir = &Directory{"/", make(map[string]*Directory), make(map[string][]*spb.VName)}
		roots[root] = dir
	}
	return dir
}

func (root *Directory) addFile(file *spb.VName) {
	path := file.GetPath()
	dir := root.ensureDir(filepath.Dir(path))
	dir.Files[filepath.Base(path)] = append(dir.Files[filepath.Base(path)], file)
}

func (root *Directory) ensureDir(path string) *Directory {
	if path == "/" || path == "" || path == "." {
		return root
	}

	parent := root.ensureDir(filepath.Dir(path))
	name := filepath.Base(path)
	if dir := parent.Dirs[name]; dir != nil {
		return dir
	}

	dir := &Directory{name, make(map[string]*Directory), make(map[string][]*spb.VName)}
	parent.Dirs[name] = dir
	return dir
}

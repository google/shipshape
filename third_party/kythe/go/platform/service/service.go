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

// Package service implements an interface to compilation unit data received
// via an implementation of the CompilationAnalyzer RPC service defined in
// kythe/proto/analysis_proto.  File data are retrieved via the
// corresponding FileDataService.
package service

import (
	"errors"
	"log"

	"code.google.com/p/goprotobuf/proto"

	"third_party/kythe/go/platform/analysis"
	"third_party/kythe/go/platform/cache"
	"third_party/kythe/go/rpc/server"
	"third_party/kythe/go/storage/files/client"

	apb "third_party/kythe/proto/analysis_proto"
	spb "third_party/kythe/proto/storage_proto"
)

// A Service implements the CompilationAnalyzer RPC service.  It also
// implements the analysis.Fetcher and analysis.Sink interfaces, via the RPC
// interface.  An empty Service is ready to use, but performs no analysis.
type Service struct {
	// DefaultFD is the default FileData that will be used if no file
	// data service address is received in the AnalysisRequest.  If this is
	// nil, an error will result when attempting to fetch any file data for a
	// request that does not provide its own data service address.
	DefaultFD *client.FileData

	// Analyzer performs the desired analysis for each request.  If this is
	// nil, no analysis is performed, but the server will report success.
	Analyzer analysis.Analyzer

	// Cache is used to cache file data locally.  If nil, no data are cached.
	Cache *cache.Cache
}

// Analyze implements the corresponding method of the CompilationAnalyzer service.
func (s *Service) Analyze(ctx server.Context, in *apb.AnalysisRequest, stream chan<- *spb.Entry) error {
	log.Printf("Analysis requested for target %q", in.GetCompilation().GetVName().GetSignature())

	unit := &compilation{
		request: in,
		stream:  stream,
		client:  s.DefaultFD, // ...provisionally
	}

	// If a data service address was provided in the request, use that instead
	// of the default.
	if addr := in.GetFileDataService(); addr != "" {
		unit.client = s.newClient(addr)
	}

	// If we get to this point, we will return success from the RPC even if
	// analysis fails, unless it's due to an RPC error.  An error from the
	// analyzer is interpreted to mean that the results are incomplete.
	err := s.Analyzer.Analyze(in, cache.Fetcher(unit, s.Cache), unit)
	log.Printf("Analysis returned error value [%+v]", err)
	size, hits, misses := s.Cache.Stats()
	log.Printf("Cache status resident=%d hits=%d misses=%d", size, hits, misses)

	return nil
}

// newClient returns a new client communicating with the FileDataService at addr.
func (s *Service) newClient(addr string) *client.FileData {
	return client.New(addr)
}

// compilation implements the analysis.Fetcher and analysis.Sink interfaces for
// analyzing a single compilation.
type compilation struct {
	request *apb.AnalysisRequest // Original analysis request
	stream  chan<- *spb.Entry    // Stream of entries
	client  *client.FileData
}

// ErrNoFileDataService is returned by Fetch when there was no file data
// service available to query for a requested file.
var ErrNoFileDataService = errors.New("no file data service is available")

// Fetch implements the analysis.Fetcher interface.  Returns
// ErrNoFileDataService if there is no data service available in the
// compilation.
func (c *compilation) Fetch(path, digest string) ([]byte, error) {
	if c.client == nil {
		return nil, ErrNoFileDataService
	}

	return c.client.FileData(path, digest)
}

// WriteBytes implements the analysis.Sink interface.  This implementation
// returns an error only if data is not a spb.Entry protobuf.
func (c *compilation) WriteBytes(data []byte) error {
	var entry spb.Entry
	if err := proto.Unmarshal(data, &entry); err != nil {
		return err
	}

	c.stream <- &entry
	return nil
}

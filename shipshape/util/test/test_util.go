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

package test

import (
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"shipshape/util/rpc/server"

	"github.com/golang/protobuf/proto"

	notepb "github.com/google/shipshape/shipshape/proto/note_proto"
	ctxpb "github.com/google/shipshape/shipshape/proto/shipshape_context_proto"
	rpcpb "github.com/google/shipshape/shipshape/proto/shipshape_rpc_proto"
)

type Analyzer interface {
	Analyze(*ctxpb.ShipshapeContext) ([]*notepb.Note, error)
}

// CreateContext returns a new Shipshape test context
func CreateContext(testDir string, filepaths []string) (*ctxpb.ShipshapeContext, error) {
	context := &ctxpb.ShipshapeContext{
		FilePath: filepaths,
		RepoRoot: proto.String(testDir),
	}
	return context, nil
}

func CreateLocation(path string) *notepb.Location {
	return &notepb.Location{
		Path: proto.String(path),
	}
}

// RunAnalyzer does any preparation for running the analyzer, and runs
// it. Errors from setup/teardown will cause the test t to stop. Errors from the
// analysis will be returned, along with the notes
func RunAnalyzer(ctx *ctxpb.ShipshapeContext, a Analyzer, t *testing.T) ([]*notepb.Note, error) {
	oDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Could not get the current directory: %v", err)
	}

	if err := os.Chdir(*ctx.RepoRoot); err != nil {
		t.Fatalf("Could not change into test directory %s: %v", ctx.GetRepoRoot(), err)
	}

	defer os.Chdir(oDir)

	return a.Analyze(ctx)
}

// ChangeIntoTestDir changes into test directory and returns the original.
// Note: make sure to call defer os.Chdir(original) after using this!
func ChangeIntoTestDir(testDir string) (string, error) {
	oDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if err = os.Chdir(testDir); err != nil {
		return "", err
	}
	return oDir, nil
}

// CheckNoteContainsContent checks that there is a 1:1 mapping from expected
// notes to actual notes. The description in the execpted note needs only to be
// a substring of the actual note. Returns whether such a 1:1 mapping exists,
// and if not, a message to print to the tests of the unmatched results
func CheckNoteContainsContent(expected []*notepb.Note, actual []*notepb.Note) (bool, string) {
	var unmatched []*notepb.Note
	remaining := append([]*notepb.Note{}, actual...)
	var found bool

	for _, expect := range expected {
		found = false
		for i, act := range remaining {
			if Match(expect, act) {
				remaining = append(remaining[:i], remaining[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			unmatched = append(unmatched, expect)
		}
	}
	if len(unmatched) == 0 && len(remaining) == 0 {
		return true, ""
	}
	return false, fmt.Sprintf("Had unmatched results.\nExpect: %v\nActual: %v", unmatched, remaining)
}

// CheckFailureContainsContent checks that there is a 1:1 mapping from expected
// failures to actual faulres. The description in the execpted failure needs
// only to be a substring of the actual failure. Returns whether such a 1:1
// mapping exists, and if not, a message to print to the tests of the unmatched
// results
func CheckFailureContainsContent(expected []*rpcpb.AnalysisFailure, actual []*rpcpb.AnalysisFailure) (bool, string) {
	var unmatched []*rpcpb.AnalysisFailure
	remaining := append([]*rpcpb.AnalysisFailure{}, actual...)
	var found bool

	for _, expect := range expected {
		found = false
		for i, act := range remaining {
			if MatchFailure(expect, act) {
				remaining = append(remaining[:i], remaining[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			unmatched = append(unmatched, expect)
		}
	}
	if len(unmatched) == 0 && len(remaining) == 0 {
		return true, ""
	}
	return false, fmt.Sprintf("Had unmatched results.\nExpect: %v\nActual: %v", unmatched, remaining)
}

// Match determines whether the actual note matches our expected note.
// It requires that the category, subcategory, and location.path be the same, and
// it requires that the actual description contain the expected description (to make
// tests easier to write, the actual description is allowed to have more text.
// Other fields are not checked.
func Match(expect *notepb.Note, actual *notepb.Note) bool {
	// Check that the category matches what we expect.
	// Can't send nil into Category at all.
	if actual.Category == nil || *expect.Category != *actual.Category {
		return false
	}

	// Check that the subcategory matches what we expect
	if expect.Subcategory == nil && actual.Subcategory != nil {
		return false
	}
	if expect.Subcategory != nil && (actual.Subcategory == nil || *expect.Subcategory != *actual.Subcategory) {
		return false
	}

	// If we want to check location, go ahead and check it.
	// If expect.Location is nil, let any location be allowed.
	if expect.Location != nil {
		if actual.Location == nil {
			return false
		}
		if expect.Location.Path != nil && *expect.Location.Path != *actual.Location.Path {
			return false
		}
	}
	// TODO(ciera): Add other checks for the proto, like the location check above.

	return strings.Contains(*actual.Description, *expect.Description)
}

// MatchFailure determines whether the actual failure matches our expected failure.
// It requires that the category be the same, and
// it requires that the actual description contain the expected description (to make
// tests easier to write, the actual description is allowed to have more text.
func MatchFailure(expect *rpcpb.AnalysisFailure, actual *rpcpb.AnalysisFailure) bool {
	// Check that the category matches what we expect.
	// Can't send nil into Category at all.
	if actual.Category == nil || *expect.Category != *actual.Category {
		return false
	}

	return strings.Contains(*actual.FailureMessage, *expect.FailureMessage)
}

func CreatekRPCTestServer(dispatcher interface{}, name string) (string, func(), error) {
	kRPCServer := server.Service{Name: name}
	if err := kRPCServer.Register(dispatcher); err != nil {
		return "", nil, err
	}
	testServer := httptest.NewServer(server.Endpoint{&kRPCServer})
	cleanup := func() {
		testServer.CloseClientConnections()
		testServer.Close()
	}

	return testServer.URL, cleanup, nil
}

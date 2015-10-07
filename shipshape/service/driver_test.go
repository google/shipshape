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

package service

import (
	"fmt"
	//	"path/filepath"
	"reflect"
	"strings"
	"testing"

	strset "github.com/google/shipshape/shipshape/util/strings"
	"shipshape/util/rpc/server"
	testutil "shipshape/util/test"

	"github.com/golang/protobuf/proto"

	notepb "github.com/google/shipshape/shipshape/proto/note_proto"
	ctxpb "github.com/google/shipshape/shipshape/proto/shipshape_context_proto"
	rpcpb "github.com/google/shipshape/shipshape/proto/shipshape_rpc_proto"
)

type fakeDispatcher struct {
	categories []string
	files      []string
}

func (f fakeDispatcher) GetCategory(ctx server.Context, in *rpcpb.GetCategoryRequest) (*rpcpb.GetCategoryResponse, error) {
	return &rpcpb.GetCategoryResponse{
		Category: f.categories,
	}, nil
}

func (f fakeDispatcher) GetStage(ctx server.Context, in *rpcpb.GetStageRequest) (*rpcpb.GetStageResponse, error) {
	return &rpcpb.GetStageResponse{
		Stage: ctxpb.Stage_PRE_BUILD.Enum(),
	}, nil
}
func isSubset(a, b strset.Set) bool {
	return len(a.Intersect(b)) == len(a)
}

func (f fakeDispatcher) Analyze(ctx server.Context, in *rpcpb.AnalyzeRequest) (*rpcpb.AnalyzeResponse, error) {
	var nts []*notepb.Note

	// Assert that the analyzer was called with the right categories.
	if !isSubset(strset.New(in.Category...), strset.New(f.categories...)) {
		return nil, fmt.Errorf("Category mismatch: got %v, supports %v", in.Category, f.categories)
	}

	for _, file := range f.files {
		nts = append(nts, &notepb.Note{
			Category:    proto.String(f.categories[0]),
			Description: proto.String("Hello world"),
			Location:    testutil.CreateLocation(file),
		})
	}

	return &rpcpb.AnalyzeResponse{
		Note: nts,
	}, nil
}

type errDispatcher struct{}

func (errDispatcher) GetCategory(ctx server.Context, in *rpcpb.GetCategoryRequest) (*rpcpb.GetCategoryResponse, error) {
	return nil, fmt.Errorf("An error")
}

func (errDispatcher) GetStage(ctx server.Context, in *rpcpb.GetStageRequest) (*rpcpb.GetStageResponse, error) {
	return nil, fmt.Errorf("An error")
}

type panicDispatcher struct{}

func (panicDispatcher) GetCategory(ctx server.Context, in *rpcpb.GetCategoryRequest) (*rpcpb.GetCategoryResponse, error) {
	panic("panic")
}

func (panicDispatcher) GetStage(ctx server.Context, in *rpcpb.GetStageRequest) (*rpcpb.GetStageResponse, error) {
	panic("panic")
}

type fullFakeDispatcher struct {
	response *rpcpb.AnalyzeResponse
}

func (f fullFakeDispatcher) Analyze(ctx server.Context, in *rpcpb.AnalyzeRequest) (*rpcpb.AnalyzeResponse, error) {
	return f.response, nil
}

func TestGetServiceInfo(t *testing.T) {
	addr2, cleanup, err := testutil.CreatekRPCTestServer(&fakeDispatcher{[]string{"Foo", "Bar"}, nil}, "AnalyzerService")
	if err != nil {
		t.Fatalf("Registering analyzer service failed: %v", err)
	}
	defer cleanup()

	addr0, cleanup, err := testutil.CreatekRPCTestServer(&fakeDispatcher{nil, nil}, "AnalyzerService")
	if err != nil {
		t.Fatalf("Registering analyzer service failed: %v", err)
	}
	defer cleanup()

	addre, cleanup, err := testutil.CreatekRPCTestServer(&errDispatcher{}, "AnalyzerService")
	if err != nil {
		t.Fatalf("Registering analyzer service failed: %v", err)
	}
	defer cleanup()

	addrp, cleanup, err := testutil.CreatekRPCTestServer(&panicDispatcher{}, "AnalyzerService")
	if err != nil {
		t.Fatalf("Registering analyzer service failed: %v", err)
	}
	defer cleanup()

	tests := []struct {
		addrs  []string
		result map[string]strset.Set
	}{
		{[]string{addr0}, map[string]strset.Set{addr0: nil}},
		{[]string{addr2}, map[string]strset.Set{addr2: strset.New("Foo", "Bar")}},
		{[]string{addr0, addr2}, map[string]strset.Set{addr2: strset.New("Foo", "Bar"), addr0: nil}},
		{[]string{addrp, addr2}, map[string]strset.Set{addr2: strset.New("Foo", "Bar"), addrp: nil}},
		{[]string{addre, addr2}, map[string]strset.Set{addr2: strset.New("Foo", "Bar"), addre: nil}},
	}

	for _, test := range tests {
		driver := NewDriver(test.addrs)
		info := driver.getAllServiceInfo()

		if len(test.result) != len(info) {
			t.Errorf("Incorrect number of results: got %v, want %v", info, test.result)
		}

		for addr, expectCats := range test.result {
			if !strset.Equal(info[strings.TrimPrefix(addr, "http://")].categories.ToSlice(), expectCats.ToSlice()) {
				t.Errorf("Incorrect categories for %s: got %v, want %v", addr, info[addr].categories, expectCats)
			}
		}
	}
}

func TestCallAllAnalyzers(t *testing.T) {
	dispatcher := &fakeDispatcher{categories: []string{"Foo", "Bar"}, files: []string{"dir1/A.h", "dir1/A.cc"}}
	addr, cleanup, err := testutil.CreatekRPCTestServer(dispatcher, "AnalyzerService")
	if err != nil {
		t.Fatalf("Registering analyzer service failed: %v", err)
	}
	defer cleanup()

	driver := NewTestDriver([]serviceInfo{
		serviceInfo{addr, strset.New("Foo", "Bar"), ctxpb.Stage_PRE_BUILD},
	})

	tests := []struct {
		files      []string
		categories []string
		expect     []*notepb.Note
	}{
		{
			// - SomeOtherCategory should be dropped; if it weren't fakeDispatcher would return an error.
			// - fakeDispatcher produces notes for files we didn't ask about; they should be dropped.
			[]string{"dir1/A.cc"},
			[]string{"Foo", "SomeOtherCategory"},
			[]*notepb.Note{
				&notepb.Note{
					Category:    proto.String("Foo"),
					Description: proto.String(""),
					Location:    testutil.CreateLocation("dir1/A.cc"),
				},
			},
		},
	}
	for _, test := range tests {
		ctx := &ctxpb.ShipshapeContext{FilePath: test.files}

		ars := driver.callAllAnalyzers(strset.New(test.categories...), ctx, ctxpb.Stage_PRE_BUILD)
		var notes []*notepb.Note

		for _, ar := range ars {
			notes = append(notes, ar.Note...)
			if len(ar.Failure) > 0 {
				t.Errorf("Received failures from analyze call: %v", ar.Failure)
			}
		}

		ok, results := testutil.CheckNoteContainsContent(test.expect, notes)
		if !ok {
			t.Errorf("Incorrect notes for categories %v: %s\n got %v, want %v", test.categories, results, notes, test.expect)
		}
	}
}

func TestCallAllAnalyzersErrorCases(t *testing.T) {
	ctx := &ctxpb.ShipshapeContext{FilePath: []string{"dir1/A", "dir2/B"}}

	tests := []struct {
		response      *rpcpb.AnalyzeResponse
		expectNotes   []*notepb.Note
		expectFailure []*rpcpb.AnalysisFailure
	}{
		{ //analysis had a failure
			&rpcpb.AnalyzeResponse{
				Failure: []*rpcpb.AnalysisFailure{
					&rpcpb.AnalysisFailure{
						Category:       proto.String("Foo"),
						FailureMessage: proto.String("badbadbad"),
					},
				},
			},
			nil,
			[]*rpcpb.AnalysisFailure{
				&rpcpb.AnalysisFailure{
					Category:       proto.String("Foo"),
					FailureMessage: proto.String("badbadbad"),
				},
			},
		},
		{ //analysis had both failure and notes
			&rpcpb.AnalyzeResponse{
				Note: []*notepb.Note{
					&notepb.Note{
						Category:    proto.String("Foo"),
						Description: proto.String("A note"),
						Location:    testutil.CreateLocation("dir1/A"),
					},
					&notepb.Note{
						Category:    proto.String("Foo"),
						Description: proto.String("A note"),
						Location:    testutil.CreateLocation("dir1/A"),
					},
				},
				Failure: []*rpcpb.AnalysisFailure{
					&rpcpb.AnalysisFailure{
						Category:       proto.String("Foo"),
						FailureMessage: proto.String("badbadbad"),
					},
				},
			},
			[]*notepb.Note{
				&notepb.Note{
					Category:    proto.String("Foo"),
					Description: proto.String("A note"),
					Location:    testutil.CreateLocation("dir1/A"),
				},
				&notepb.Note{
					Category:    proto.String("Foo"),
					Description: proto.String("A note"),
					Location:    testutil.CreateLocation("dir1/A"),
				},
			},
			[]*rpcpb.AnalysisFailure{
				&rpcpb.AnalysisFailure{
					Category:       proto.String("Foo"),
					FailureMessage: proto.String("badbadbad"),
				},
			},
		},
	}
	for _, test := range tests {
		addr, cleanup, err := testutil.CreatekRPCTestServer(&fullFakeDispatcher{test.response}, "AnalyzerService")
		if err != nil {
			t.Fatalf("Registering analyzer service failed: %v", err)
		}
		defer cleanup()

		driver := NewTestDriver([]serviceInfo{
			serviceInfo{addr, strset.New("Foo"), ctxpb.Stage_PRE_BUILD},
		})

		ars := driver.callAllAnalyzers(strset.New("Foo"), ctx, ctxpb.Stage_PRE_BUILD)
		var notes []*notepb.Note
		var failures []*rpcpb.AnalysisFailure

		for _, ar := range ars {
			notes = append(notes, ar.Note...)
			failures = append(failures, ar.Failure...)
		}

		ok, results := testutil.CheckNoteContainsContent(test.expectNotes, notes)
		if !ok {
			t.Errorf("Incorrect notes for original response %v: %s\n got %v, want %v", test.response, results, notes, test.expectNotes)
		}
		ok, results = testutil.CheckFailureContainsContent(test.expectFailure, failures)
		if !ok {
			t.Errorf("Incorrect failures for original response %v: %s\n got %v, want %v", test.response, results, failures, test.expectFailure)
		}

	}
}

func TestFilterPaths(t *testing.T) {
	tests := []struct {
		label         string
		ignoreDirs    []string
		inputFiles    []string
		expectedFiles []string
	}{
		{
			"Empty ignore list",
			[]string{},
			[]string{"a", "b", "c", "d"},
			[]string{"a", "b", "c", "d"},
		},
		{
			"Ignore list matches some items",
			[]string{"dir1/", "dir2/"},
			[]string{"dir1/a", "dir3/a", "dir2/a"},
			[]string{"dir3/a"},
		},
		{
			"Ignore list does not match anything",
			[]string{"dir77/"},
			[]string{"dir1/a", "dir3/a", "dir2/a"},
			[]string{"dir1/a", "dir3/a", "dir2/a"},
		},
		{
			"Ignore list contains substring match",
			[]string{"dir1/"},
			[]string{"dir4/a", "dir4/dir1/a"},
			[]string{"dir4/a", "dir4/dir1/a"},
		},
		{
			"Ignore list matches everything",
			[]string{"dir1/"},
			[]string{"dir1/a", "dir1/b", "dir1/c"},
			nil,
		},
	}

	for _, test := range tests {
		out := filterPaths(test.ignoreDirs, test.inputFiles)
		if !reflect.DeepEqual(out, test.expectedFiles) {
			t.Errorf("Error on %q: got %v, expected %v", test.label, out, test.expectedFiles)
		}
	}
}

/*
func TestFindCompilationUnitsGood(t *testing.T) {
	tests := []struct {
		dir           string
		expectedPaths []string
	}{
		{
			"valid_kindex",
			[]string{"shipshape/service/testdata/service_test/valid_kindex/2b1a94f4695c38fd074cb80cb8a49c4951b8a12e773073e6dd180d3ffa3fbdfe.kindex"},
		},
		{
			"no_kindex",
			[]string{},
		},
	}

	for _, test := range tests {
		out, err := findCompilationUnits(filepath.Join("shipshape/service/testdata/service_test", test.dir))

		if err != nil {
			t.Errorf("Received error from directory %s: %v", test.dir, err.Error())
		}

		paths := make([]string, 0, len(out))
		for p, cu := range out {
			paths = append(paths, p)
			if cu == nil {
				t.Errorf("Received no compilation unit for path %s in %s", p, test.dir)
			}
		}
		if !strset.Equal(test.expectedPaths, paths) {
			t.Errorf("Did not get the right paths for %s: got %v, want %v", test.dir, paths, test.expectedPaths)
		}
	}
}

func TestFindCompilationUnitsError(t *testing.T) {
	tests := []string{
		"invalid_kindex",
		"invalid_dir",
	}

	for _, test := range tests {
		_, err := findCompilationUnits(filepath.Join("shipshape/service/testdata/service_test", test))

		if err == nil {
			t.Errorf("Expected an error, but did not get one for directory %s", test)
		}
	}
}
*/

package service

import (
	"fmt"
	"reflect"
	"testing"

	"third_party/kythe/go/rpc/server"
	testutil "shipshape/test"
	strset "shipshape/util/strings"

	"code.google.com/p/goprotobuf/proto"

	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
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

type panicDispatcher struct{}

func (panicDispatcher) GetCategory(ctx server.Context, in *rpcpb.GetCategoryRequest) (*rpcpb.GetCategoryResponse, error) {
	panic("panic")
}

type fullFakeDispatcher struct {
	response *rpcpb.AnalyzeResponse
}

func (f fullFakeDispatcher) Analyze(ctx server.Context, in *rpcpb.AnalyzeRequest) (*rpcpb.AnalyzeResponse, error) {
	return f.response, nil
}

func TestGetCategories(t *testing.T) {
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
		categories := driver.getAllCategories()

		if len(test.result) != len(categories) {
			t.Errorf("Incorrect number of results: got %v, want %v", categories, test.result)
		}

		for addr, cats := range test.result {
			if !strset.Equal(cats.ToSlice(), test.result[addr].ToSlice()) {
				t.Errorf("Incorrect categories for %s: got %v, want %v", addr, cats, test.result[addr])
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

	driver := NewTestDriver(map[string]strset.Set{addr: strset.New("Foo", "Bar")})

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
		cfg := &config{categories: test.categories}
		ctx := &ctxpb.ShipshapeContext{FilePath: test.files}

		ars := driver.callAllAnalyzers(cfg, ctx)
		var notes []*notepb.Note

		for _, ar := range ars {
			notes = append(notes, ar.Note...)
			if len(ar.Failure) > 0 {
				t.Errorf("Received failures from analyze call: %v", ar.Failure)
			}
		}

		ok, results := testutil.CheckNoteContainsContent(test.expect, notes)
		if !ok {
			t.Errorf("Incorrect notes for config %v: %s\n got %v, want %v", cfg, results, notes, test.expect)
		}
	}
}

func TestCallAllAnalyzersErrorCases(t *testing.T) {
	ctx := &ctxpb.ShipshapeContext{FilePath: []string{"dir1/A", "dir2/B"}}
	cfg := &config{categories: []string{"Foo"}}

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

		driver := NewTestDriver(map[string]strset.Set{addr: strset.New("Foo")})

		ars := driver.callAllAnalyzers(cfg, ctx)
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
			t.Fatalf("Error on %q: got %v, expected %v", test.label, out, test.expectedFiles)
		}
	}
}

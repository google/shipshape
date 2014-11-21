package api

import (
	"testing"

	"shipshape/util/strings"

	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
)

type fakeAnalyzer struct {
	category string
	notes    []*notepb.Note
	err      error
}

func (f fakeAnalyzer) Category() string                                        { return f.category }
func (f fakeAnalyzer) Analyze(*ctxpb.ShipshapeContext) ([]*notepb.Note, error) { return f.notes, f.err }

func TestGetCategory(t *testing.T) {
	tests := []struct {
		analyzers []Analyzer
		expected  []string
	}{
		{nil, nil},
		{[]Analyzer{fakeAnalyzer{"Foo", nil, nil}}, []string{"Foo"}},
		{[]Analyzer{fakeAnalyzer{"", nil, nil}}, []string{""}},
		{
			[]Analyzer{
				fakeAnalyzer{"Foo", nil, nil},
				fakeAnalyzer{"Bar", nil, nil},
			},
			[]string{"Foo", "Bar"},
		},
	}

	for _, test := range tests {
		a := CreateAnalyzerService(test.analyzers)

		in := &rpcpb.GetCategoryRequest{}
		resp, _ := a.GetCategory(nil, in)

		if !strings.Equal(resp.Category, test.expected) {
			t.Errorf("For analyzers %v: got %v, want %v", test.analyzers, resp.Category, test.expected)
		}
	}
}

// TODO(ciera): test analyze!

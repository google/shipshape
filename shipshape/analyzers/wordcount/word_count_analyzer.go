package wordcount

import (
	"fmt"
	"io/ioutil"
	"strings"

	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"

	"code.google.com/p/goprotobuf/proto"
)

type WordCountAnalyzer struct {
}

func (WordCountAnalyzer) Category() string { return "WordCount" }

func (p WordCountAnalyzer) Analyze(ctx *ctxpb.ShipshapeContext) ([]*notepb.Note, error) {
	var notes []*notepb.Note
	notes = make([]*notepb.Note, len(ctx.FilePath))
	for i, path := range ctx.FilePath {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("could not get file contents for %s: %v", path, err)
		}
		content := string(bytes)
		count := p.CountWords(content)
		notes[i] = &notepb.Note{
			Location: &notepb.Location{
				Path:          proto.String(path),
				SourceContext: ctx.SourceContext,
			},
			Category:    proto.String(p.Category()),
			Description: proto.String(fmt.Sprintf("Word count: %v", count)),
		}
	}
	return notes, nil
}

// CountWords returns the number of words found in content
func (WordCountAnalyzer) CountWords(content string) int {
	return len(strings.Fields(content))
}

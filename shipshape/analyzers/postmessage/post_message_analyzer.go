package postmessage

import (
	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"

	"code.google.com/p/goprotobuf/proto"
)

type PostMessageAnalyzer struct {
}

func (p PostMessageAnalyzer) Category() string {
	return "PostMessage"
}

func (p PostMessageAnalyzer) Analyze(ctx *ctxpb.ShipshapeContext) ([]*notepb.Note, error) {
	note := &notepb.Note{
		Category:    proto.String(p.Category()),
		Description: proto.String("Shipshape analysis results are being produced"),
		Location:    &notepb.Location{SourceContext: ctx.SourceContext},
		Severity:    notepb.Note_OTHER.Enum(),
	}
	var notearray = []*notepb.Note{note}
	return notearray, nil
}

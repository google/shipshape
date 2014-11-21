// Package api provides functionality for implementing Shipshape analyzers.
package api

import (
	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
)

// An Analyzer provides the shipshape service with the functionality to run analysis
// from the various environments that Shipshape can run in.
type Analyzer interface {
	// A Category describes this specific analysis.
	// Should not contain spaces or other special characters.
	Category() string

	// Analyze runs this analyzer's analysis.
	// Returns a list of Finding protos for any issues found.
	// Returns an error for any problems with running the analysis. In cases
	// where there is an error, there can be partial results in the notes.
	// Before analyzing, this method should check the ShipshapeContext to
	// see if it needs to analyze at all, and should return quickly in
	// that case.
	Analyze(*ctxpb.ShipshapeContext) ([]*notepb.Note, error)
}

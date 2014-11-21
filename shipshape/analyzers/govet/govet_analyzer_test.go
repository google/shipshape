package govet

import (
	"testing"

	ctxpb "shipshape/proto/shipshape_context_proto"
)

const (
	noErrors  = "shipshape/analyzers/govet/testdata/no_errors.go"
	hasErrors = "shipshape/analyzers/govet/testdata/has_errors.go"
)

func TestNoErrorsInGoFile(t *testing.T) {
	ctx := &ctxpb.ShipshapeContext{}
	gva := new(GoVetAnalyzer)

	notes, err := gva.analyzeOneFile(ctx, noErrors)
	if err != nil {
		t.Errorf("Analysis of %q failed: %v", noErrors, err)
	}

	if len(notes) != 0 {
		t.Errorf("Expected 0 notes, got %d: %v", len(notes), notes)
	}
}

func TestErrorsInGoFile(t *testing.T) {
	ctx := &ctxpb.ShipshapeContext{}
	gva := new(GoVetAnalyzer)

	expectedNotes := []struct {
		filePath string
		lineno   int32
		message  string
	}{
		{hasErrors, 8, "too few arguments in call to Fprintf"},
		{hasErrors, 9, "no formatting directive in Fprintf call"},
	}

	notes, err := gva.analyzeOneFile(ctx, hasErrors)
	if err != nil {
		t.Errorf("Analysis of %q failed: %v", hasErrors, err)
	}
	if len(notes) != len(expectedNotes) {
		t.Errorf("Expected %d notes, got %d: %v", len(expectedNotes), len(notes), notes)
	}

	for i, note := range notes {
		expected := expectedNotes[i]
		if *note.Location.Path != expected.filePath {
			t.Errorf("Expected path %s, got %s: %v", expected.filePath, *note.Location.Path, note)
		}
		if *note.Location.Range.StartLine != expected.lineno {
			t.Errorf("Expected start line %d, got %d: %v", expected.lineno, *note.Location.Range.StartLine, note)
		}
		if *note.Description != expected.message {
			t.Errorf("Expected message %q, got %q: %v", expected.message, *note.Description, note)
		}
	}
}

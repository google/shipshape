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

package govet

import (
	"os"
	"testing"

	ctxpb "github.com/google/shipshape/shipshape/proto/shipshape_context_proto"
)

const (
	noErrors  = "shipshape/analyzers/govet/testdata/no_errors.go"
	hasErrors = "shipshape/analyzers/govet/testdata/has_errors.go"
)

func init() {
	// Use the version of Go we know to be available via Bazel.
	goCmd = "tools/go/go"
	os.Setenv("GOROOT", "tools/go/GOROOT")
}

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
		{hasErrors, 24, "too few arguments in call to Fprintf"},
		{hasErrors, 25, "no formatting directive in Fprintf call"},
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

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

// Package api provides functionality for implementing Shipshape analyzers.
package api

import (
	notepb "github.com/google/shipshape/shipshape/proto/note_proto"
	ctxpb "github.com/google/shipshape/shipshape/proto/shipshape_context_proto"
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

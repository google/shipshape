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

package postmessage

import (
	"github.com/golang/protobuf/proto"

	notepb "github.com/google/shipshape/shipshape/proto/note_proto"
	ctxpb "github.com/google/shipshape/shipshape/proto/shipshape_context_proto"
)

type PostMessageAnalyzer struct {
}

func (p PostMessageAnalyzer) Category() string {
	return "PostMessage"
}

func (p PostMessageAnalyzer) Analyze(ctx *ctxpb.ShipshapeContext) ([]*notepb.Note, error) {
	note := &notepb.Note{
		Category:    proto.String(p.Category()),
		Description: proto.String("Shipshape analysis results are being produced from go dispatcher at stage PRE_BUILD"),
		Location:    &notepb.Location{SourceContext: ctx.SourceContext},
		Severity:    notepb.Note_OTHER.Enum(),
	}
	var notearray = []*notepb.Note{note}
	return notearray, nil
}

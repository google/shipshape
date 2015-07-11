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

package api

import (
	"log"

	"shipshape/util/file"
	strset "shipshape/util/strings"
	"shipshape/util/rpc/server"

	"github.com/golang/protobuf/proto"

	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
)

// analyzerService is a service that accepts AnalyzerRequests, as defined by shipshape_rpc_proto.
// Third parties can provide their own analyzers by creating a new service with the set of
// analyzers that they wish to run, and then starting this service using kythe/go/rpc/server.
// Third parties will need to provide an appropriate Docker image that includes the relevant
// dependencies, starts up the service, and exposes the port. It requires that all analyzers
// be run at the same stage.
type analyzerService struct {
	analyzers []Analyzer
	stage     ctxpb.Stage
}

func CreateAnalyzerService(analyzers []Analyzer, stage ctxpb.Stage) *analyzerService {
	return &analyzerService{analyzers, stage}
}

// Analyze will determine which analyzers to run and call them as appropriate. If necessary, it will
// also modify the context before calling the analyzers. It recovers from all analyzer panics with a
// note that the analyzer failed.
func (s analyzerService) Analyze(ctx server.Context, in *rpcpb.AnalyzeRequest) (resp *rpcpb.AnalyzeResponse, err error) {
	resp = new(rpcpb.AnalyzeResponse)

	log.Printf("called with: %v", proto.MarshalTextString(in))
	log.Print("starting analyzing")
	var nts []*notepb.Note
	var errs []*rpcpb.AnalysisFailure

	defer func() {
		resp.Note = nts
		resp.Failure = errs
	}()

	orgDir, restore, err := file.ChangeDir(*in.ShipshapeContext.RepoRoot)
	if err != nil {
		log.Printf("Internal error before analyzing: %v", err)
		appendFailure(&errs, "InternalFailure", err)
		return resp, err
	}
	defer func() {
		if err := restore(); err != nil {
			log.Printf("could not return back into %s from %s: %v", orgDir, *in.ShipshapeContext.RepoRoot, err)
		}
	}()

	reqCats := strset.New(in.Category...)
	for _, a := range s.analyzers {
		if reqCats.Contains(a.Category()) {
			runAnalyzer(a, in.ShipshapeContext, &nts, &errs)
		}
	}
	log.Printf("finished analyzing, sending back %d notes and %d errors", len(nts), len(errs))
	return resp, nil
}

// GetCategory gets the list of categories in this analyzer pack
func (s analyzerService) GetCategory(ctx server.Context, in *rpcpb.GetCategoryRequest) (*rpcpb.GetCategoryResponse, error) {
	var cs []string
	for _, a := range s.analyzers {
		cs = append(cs, a.Category())
	}
	return &rpcpb.GetCategoryResponse{
		Category: cs,
	}, nil
}

// GetStage returns the stage of the analyzers. All registered analyzers must have the same
// stage, otherwise this will return an error.
func (s analyzerService) GetStage(ctx server.Context, in *rpcpb.GetStageRequest) (*rpcpb.GetStageResponse, error) {
	return &rpcpb.GetStageResponse{Stage: s.stage.Enum()}, nil
}

// runAnalyzer attempts to run the given analyzer on the provided context. It returns the list of notes
// and errors that occured in the process.
func runAnalyzer(analyzer Analyzer, ctx *ctxpb.ShipshapeContext, nts *[]*notepb.Note, errs *[]*rpcpb.AnalysisFailure) {
	c := analyzer.Category()
	log.Printf("About to run analyzer: %v", c)

	notes, err := analyzer.Analyze(ctx)
	if err != nil {
		appendFailure(errs, c, err)
	}
	*nts = append(*nts, notes...)
}

// appendFailure adds a new analysis failure to the list in errs
func appendFailure(errs *[]*rpcpb.AnalysisFailure, cat string, err error) {
	*errs = append(*errs, &rpcpb.AnalysisFailure{
		Category:       proto.String(cat),
		FailureMessage: proto.String(err.Error()),
	})
}

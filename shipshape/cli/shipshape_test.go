package cli

import (
	"testing"

	rpcpb "shipshape/proto/shipshape_rpc_proto"  
)

func countFailures(resp rpcpb.ShipshapeResponse) int {
	failures := 0
	for _, analyzeResp := range resp.AnalyzeResponse {
		failures += len(analyzeResp.Failure)
	}
	return failures
}

func countNotes(resp rpcpb.ShipshapeResponse) int {
	notes := 0
	for _, analyzeResp := range resp.AnalyzeResponse {
		notes += len(analyzeResp.Note)
	}
	return notes
}

func countCategoryNotes(resp rpcpb.ShipshapeResponse, category string) int {
	notes := 0
	for _, analyzeResp := range resp.AnalyzeResponse {
		for _, note := range analyzeResp.Note {
			if *note.Category == category {
				notes += 1
			}
		}
	}
	return notes
}

func TestBasic(t *testing.T) {
	options := Options{
		File:                "shipshape/cli/testdata/workspace1",
		ThirdPartyAnalyzers: []string{},
		Build:               "maven",
		TriggerCats:         []string{"PostMessage", "JSHint"},
		Dind:                false,
		Event:               "manual", // TODO: const
		Repo:                "gcr.io/shipshape_releases", // TODO: const
		StayUp:              true,
		// TODO(rsk): current e2e test require tag to be specified on
		// the command line. Furthermore, if the tag is "local" it builds
		// containers and tags them.
		Tag:                 "prod",
		// TODO(rsk): current e2e test can be ru both with & without kythe.
		LocalKythe:          false,
	}
	var allResponses rpcpb.ShipshapeResponse 
	options.HandleResponse = func(shipshapeResp *rpcpb.ShipshapeResponse, _ string) error {
		allResponses.AnalyzeResponse = append(allResponses.AnalyzeResponse, shipshapeResp.AnalyzeResponse...)
		return nil
	}

	numNotes, err := New(options).Run()
	if err != nil {
		t.Fatal(err)
	}
	if countFailures(allResponses) != 0 {
		t.Errorf("Expected 0 failures: %v", allResponses)
	}
	if numNotes != countNotes(allResponses) {
		t.Errorf("Disagreement between returned count (%v) and count of notes: %v", numNotes, allResponses)
	}
	if numNotes != 10 {
		t.Errorf("Expected 10 notes, got %v: %v", numNotes, allResponses)
	}
	jshintNotes := countCategoryNotes(allResponses, "JSHint")
	if jshintNotes != 8 {
		t.Errorf("Expected 8 notes from JSHint, got %v: %v", jshintNotes, allResponses)
	}
	postMessageNotes := countCategoryNotes(allResponses, "PostMessage")
	if postMessageNotes != 2 {
		t.Errorf("Expected 2 notes from PostMessage, got %v: %v", postMessageNotes, allResponses)
	}
}

func TestExternalAnalyzers(t *testing.T) {
	// Replaces part of the e2e test
	// Create a fake maven project with android failures

	// Run CLI using a .shipshape file
}

func TestBuiltInAnalyzersPreBuild(t *testing.T) {
	// Replaces part of the e2e test
	// Test PostMessage and Go, with no kythe build

}

func TestBuiltInAnalyzersPostBuild(t *testing.T) {
	// Replaces part of the e2e test
	// Test with a kythe maven build
	// PostMessage and ErrorProne
}

func TestStreamsMode(t *testing.T) {
	// Test whether it works in streams mode
	// Before creating this, ensure that streams mode
	// is actually still something we need to support.
}

func TestChangingDirectories(t *testing.T) {
	// Replaces the changedir test
	// Make sure to test changing down, changing up, running on the same directory, running on a single file in the same directory, and changing to a sibling
}

func dumpLogs() {

}

func checkOutput(category string, numResults int) {

}

package cli

import (
	"flag"
	"testing"

	rpcpb "github.com/google/shipshape/shipshape/proto/shipshape_rpc_proto"  
)

var dockerTag = flag.String("shipshape_test_docker_tag", "", "the docker tag for the images to use for testing")

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

func TestExternalAnalyzers(t *testing.T) {
	// Replaces part of the e2e test
	// Create a fake maven project with android failures

	// Run CLI using a .shipshape file
}

func TestBuiltInAnalyzersPreBuild(t *testing.T) {
	options := Options{
		File:                "shipshape/cli/testdata/workspace1",
		ThirdPartyAnalyzers: []string{},
		Build:               "",
		TriggerCats:         []string{"PostMessage", "JSHint", "go vet", "PyLint"},
		Dind:                false,
		Event:               "manual", // TODO: const
		Repo:                "gcr.io/shipshape_releases", // TODO: const
		StayUp:              true,
		Tag:                 *dockerTag,
		// TODO(rsk): current e2e test can be run both with & without kythe.
		LocalKythe:          false,
	}
	var allResponses rpcpb.ShipshapeResponse
	options.HandleResponse = func(shipshapeResp *rpcpb.ShipshapeResponse, _ string) error {
		allResponses.AnalyzeResponse = append(allResponses.AnalyzeResponse, shipshapeResp.AnalyzeResponse...)
		return nil
	}
	returnedNotesCount, err := New(options).Run()
	if err != nil {
		t.Fatal(err)
	}
	testName := "TestBuiltInAnalyzerPreBuild"

	if got, want := countFailures(allResponses), 0; got != want {
		t.Errorf("%v: Wrong number of failures; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if countedNotes := countNotes(allResponses); returnedNotesCount != countedNotes {
		t.Errorf("%v: Inconsistent note count: returned %v, counted %v (proto data: %v", testName, returnedNotesCount, countedNotes, allResponses)
	}
	if got, want := returnedNotesCount, 21; got != want {
		t.Errorf("%v: Wrong number of notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if got, want := countCategoryNotes(allResponses, "PostMessage"), 2; got != want {
		t.Errorf("%v: Wrong number of PostMessage notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if got, want := countCategoryNotes(allResponses, "JSHint"), 8; got != want {
		t.Errorf("%v: Wrong number of JSHint notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if got, want := countCategoryNotes(allResponses, "go vet"), 0; got != want {
		t.Errorf("%v: Wrong number of go vet notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if got, want := countCategoryNotes(allResponses, "PyLint"), 10; got != want {
		t.Errorf("%v: Wrong number of PyLint notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
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
